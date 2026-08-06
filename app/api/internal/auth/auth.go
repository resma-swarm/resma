// Package auth implementa autenticação JWT + bcrypt + API key para a API Go.
//
// Portado de backend/services/auth.py. Mantém paridade total com o fluxo
// Python: onboarding, login, refresh, logout, change-password, get_current_user.
//
// Novidade 0b.4 (vs Python): API key model com scopes read/write. Tabela
// api_keys no DuckDB + middleware que valida Authorization: Bearer
// resma_key_... ou X-API-Key header. Endpoints /api/auth/api-keys (CRUD)
// para admin gerenciar keys via UI.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/resma-swarm/resma/app/api/internal/config"
	"github.com/resma-swarm/resma/app/api/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// Service encapsula a lógica de autenticação.
type Service struct {
	cfg *config.Config
	db  *db.Store
	log *slog.Logger

	// Rate limiting de login (in-memory, por IP)
	mu            sync.Mutex
	loginAttempts map[string][]float64
}

// New cria um novo Service de auth.
func New(cfg *config.Config, database *db.Store) *Service {
	return &Service{
		cfg:           cfg,
		db:            database,
		log:           slog.Default().With("component", "auth"),
		loginAttempts: make(map[string][]float64),
	}
}

// Claims representa os claims do JWT.
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Type     string `json:"type"`            // "access" ou "refresh"
	Token    string `json:"token,omitempty"` // apenas para refresh
	jwt.RegisteredClaims
}

// AuthResult é o resultado de login/onboarding/refresh.
type AuthResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
}

// UserContext representa o usuário autenticado (do JWT decodificado).
type UserContext struct {
	Sub      string
	Username string
	Role     string
}

// ---------------------------------------------------------------------------
// JWT secret
// ---------------------------------------------------------------------------

// getJWTSecret retorna o secret do JWT: config > app_settings > gerado.
func (s *Service) getJWTSecret(ctx context.Context) string {
	if s.cfg.JWTSecret != "" {
		return s.cfg.JWTSecret
	}
	stored, err := s.db.GetSetting(ctx, "jwt_secret")
	if err == nil && stored != "" {
		return stored
	}
	// Gerar e persistir (dev apenas — produção deve ter RESMA_JWT_SECRET)
	generated := generateUUID()
	_ = s.db.SetSetting(ctx, "jwt_secret", generated)
	s.log.Warn("JWT secret gerado automaticamente — usar apenas em dev")
	return generated
}

// ---------------------------------------------------------------------------
// bcrypt
// ---------------------------------------------------------------------------

// HashPassword gera o hash bcrypt de uma senha.
func (s *Service) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(bytes), nil
}

// VerifyPassword verifica se a senha corresponde ao hash bcrypt.
func (s *Service) VerifyPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ---------------------------------------------------------------------------
// Token generation
// ---------------------------------------------------------------------------

// CreateAccessToken gera um JWT access token para o usuário.
func (s *Service) CreateAccessToken(ctx context.Context, userID int32, username, role string) (string, error) {
	now := time.Now()
	expiry := now.Add(s.cfg.JWTAccessTTL)
	claims := &Claims{
		Username: username,
		Role:     role,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.getJWTSecret(ctx)))
}

// CreateRefreshToken gera um JWT refresh token e o persiste no DB.
func (s *Service) CreateRefreshToken(ctx context.Context, userID int32) (string, error) {
	token := generateUUID()
	expiresAt := time.Now().Add(s.cfg.JWTRefreshTTL)
	if err := s.db.SaveRefreshToken(ctx, token, userID, expiresAt); err != nil {
		return "", fmt.Errorf("save refresh token: %w", err)
	}
	now := time.Now()
	claims := &Claims{
		Type:  "refresh",
		Token: token,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return jwtToken.SignedString([]byte(s.getJWTSecret(ctx)))
}

// DecodeToken decodifica e valida um JWT.
func (s *Service) DecodeToken(ctx context.Context, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", t.Header["alg"])
		}
		return []byte(s.getJWTSecret(ctx)), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("token inválido")
	}
	return claims, nil
}

// ---------------------------------------------------------------------------
// Rate limiting
// ---------------------------------------------------------------------------

// CheckRateLimit verifica se o IP pode tentar login (max N tentativas por minuto).
func (s *Service) CheckRateLimit(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := float64(time.Now().Unix())
	attempts := s.loginAttempts[ip]
	// Filtra tentativas antigas (> 60s)
	filtered := attempts[:0]
	for _, t := range attempts {
		if now-t < 60 {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= s.cfg.LoginRateLimit {
		s.loginAttempts[ip] = filtered
		return false
	}
	filtered = append(filtered, now)
	s.loginAttempts[ip] = filtered
	return true
}

// ---------------------------------------------------------------------------
// Auth flows (onboarding, login, refresh, logout, change-password)
// ---------------------------------------------------------------------------

// DoOnboarding cria o primeiro usuário owner (se nenhum existir).
// Fase 8: role mudou de 'admin' para 'owner' (owner é único e tem acesso total).
func (s *Service) DoOnboarding(ctx context.Context, username, password string) (*AuthResult, error) {
	count, err := s.db.GetUserCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user count: %w", err)
	}
	if count > 0 {
		return nil, ErrOnboardingCompleted
	}
	hash, err := s.HashPassword(password)
	if err != nil {
		return nil, err
	}
	userID, err := s.db.CreateUserWithRole(ctx, username, hash, string(RoleOwner), "")
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	access, err := s.CreateAccessToken(ctx, userID, username, string(RoleOwner))
	if err != nil {
		return nil, err
	}
	refresh, err := s.CreateRefreshToken(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "bearer",
	}, nil
}

// DoLogin autentica um usuário e retorna tokens.
func (s *Service) DoLogin(ctx context.Context, username, password, ip string) (*AuthResult, error) {
	if !s.CheckRateLimit(ip) {
		return nil, ErrTooManyAttempts
	}
	user, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if !s.VerifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	access, err := s.CreateAccessToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}
	refresh, err := s.CreateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "bearer",
	}, nil
}

// DoRefresh valida um refresh token e emite um novo access token.
func (s *Service) DoRefresh(ctx context.Context, refreshTokenStr string) (*AuthResult, error) {
	claims, err := s.DecodeToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Type != "refresh" {
		return nil, ErrInvalidTokenType
	}
	rt, err := s.db.GetRefreshToken(ctx, claims.Token)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if rt == nil {
		return nil, ErrRefreshTokenNotFound
	}
	if rt.Revoked {
		return nil, ErrRefreshTokenRevoked
	}
	user, err := s.db.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	access, err := s.CreateAccessToken(ctx, user.ID, user.Username, user.Role)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken: access,
		TokenType:   "bearer",
	}, nil
}

// DoLogout revoga o refresh token.
func (s *Service) DoLogout(ctx context.Context, refreshTokenStr string) error {
	claims, err := s.DecodeToken(ctx, refreshTokenStr)
	if err != nil {
		return nil // ignora erro — logout é idempotente
	}
	if claims.Type == "refresh" {
		_ = s.db.RevokeRefreshToken(ctx, claims.Token)
	}
	return nil
}

// DoChangePassword troca a senha do usuário e revoga todos os tokens.
func (s *Service) DoChangePassword(ctx context.Context, userID int32, currentPassword, newPassword string) error {
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	// Buscar hash da senha via GetUserByUsername (que retorna password_hash)
	fullUser, err := s.db.GetUserByUsername(ctx, user.Username)
	if err != nil {
		return fmt.Errorf("get full user: %w", err)
	}
	if fullUser == nil {
		return ErrUserNotFound
	}
	if !s.VerifyPassword(currentPassword, fullUser.PasswordHash) {
		return ErrInvalidCredentials
	}
	newHash, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.db.UpdateUserPassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	_ = s.db.RevokeAllUserTokens(ctx, userID)
	return nil
}

// ValidateAccessToken valida um access token e retorna o UserContext.
func (s *Service) ValidateAccessToken(ctx context.Context, tokenStr string) (*UserContext, error) {
	claims, err := s.DecodeToken(ctx, tokenStr)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Type != "access" {
		return nil, ErrInvalidTokenType
	}
	return &UserContext{
		Sub:      claims.Subject,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

// ---------------------------------------------------------------------------
// API Key (0b.4 — novo)
// ---------------------------------------------------------------------------

// APIKeyPrefix é o prefixo de todas as API keys geradas.
const APIKeyPrefix = "resma_key_"

// GenerateAPIKey gera uma nova API key aleatória e retorna (key_plaintext, key_hash, key_prefix).
// O plaintext só é retornado uma vez — apenas o hash é armazenado.
func GenerateAPIKey() (plaintext, hash, prefix string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("generate api key: %w", err)
	}
	plaintext = APIKeyPrefix + hex.EncodeToString(bytes)
	hash = HashAPIKey(plaintext)
	prefix = plaintext[:12] + "..." // resma_key_XX...
	return plaintext, hash, prefix, nil
}

// HashAPIKey gera o hash SHA-256 de uma API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// CreateAPIKey cria uma nova API key no DB e retorna o plaintext + id.
func (s *Service) CreateAPIKey(ctx context.Context, name, scopes string) (string, int32, error) {
	plaintext, hash, prefix, err := GenerateAPIKey()
	if err != nil {
		return "", 0, err
	}
	id, err := s.db.CreateAPIKey(ctx, hash, prefix, name, scopes)
	if err != nil {
		return "", 0, fmt.Errorf("create api key: %w", err)
	}
	return plaintext, id, nil
}

// ValidateAPIKey valida uma API key e retorna a key record se válida.
func (s *Service) ValidateAPIKey(ctx context.Context, key string) (*db.APIKey, error) {
	if !strings.HasPrefix(key, APIKeyPrefix) {
		return nil, ErrInvalidAPIKey
	}
	hash := HashAPIKey(key)
	apiKey, err := s.db.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	if apiKey == nil {
		return nil, ErrInvalidAPIKey
	}
	// Atualizar last_used_at (best-effort, não bloqueia)
	_ = s.db.UpdateAPIKeyLastUsed(ctx, apiKey.ID)
	return apiKey, nil
}

// HasScope verifica se a API key tem um scope específico.
func HasScope(apiKey *db.APIKey, scope string) bool {
	scopes := strings.Split(apiKey.Scopes, ",")
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == scope || s == "write" { // write implica read
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// generateUUID gera um UUID v4 aleatório (sem dependência externa).
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---------------------------------------------------------------------------
// Erros sentinela
// ---------------------------------------------------------------------------

var (
	ErrOnboardingCompleted  = fmt.Errorf("onboarding already completed")
	ErrTooManyAttempts      = fmt.Errorf("too many login attempts")
	ErrInvalidCredentials   = fmt.Errorf("usuário ou senha inválidos")
	ErrInvalidToken         = fmt.Errorf("invalid token")
	ErrInvalidTokenType     = fmt.Errorf("invalid token type")
	ErrRefreshTokenNotFound = fmt.Errorf("refresh token not found")
	ErrRefreshTokenRevoked  = fmt.Errorf("refresh token revoked")
	ErrUserNotFound         = fmt.Errorf("user not found")
	ErrInvalidAPIKey        = fmt.Errorf("invalid api key")
)

// HTTP status codes para os erros sentinela.
func ErrorToStatus(err error) int {
	switch err {
	case ErrOnboardingCompleted:
		return http.StatusForbidden
	case ErrTooManyAttempts:
		return http.StatusTooManyRequests
	case ErrInvalidCredentials, ErrInvalidToken, ErrInvalidTokenType,
		ErrRefreshTokenNotFound, ErrRefreshTokenRevoked, ErrInvalidAPIKey:
		return http.StatusUnauthorized
	case ErrUserNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
