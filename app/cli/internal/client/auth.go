// Package client — funções de autenticação (login, refresh, logout).
package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LoginResult é a resposta do POST /api/auth/login.
type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// Login autentica com username/password no servidor e persiste as credenciais.
// Retorna as credenciais recém-criadas.
func Login(ctx context.Context, serverURL, username, password string) (*Credentials, error) {
	api := NewWithURL(serverURL)

	var result LoginResult
	body := map[string]string{
		"username": username,
		"password": password,
	}
	if err := api.PostUnauth(ctx, "/api/auth/login", body, &result); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	creds := &Credentials{
		ServerURL:    serverURL,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		Username:     username,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}

	// Tentar extrair role do token (opcional — não falha se não conseguir)
	creds.Role = parseRoleFromToken(result.AccessToken)

	if err := SaveCredentials(creds); err != nil {
		return nil, fmt.Errorf("save credentials: %w", err)
	}

	return creds, nil
}

// Logout limpa as credenciais persistidas (remove o arquivo).
// Não chama o servidor — o JWT stateless não tem server-side logout.
func Logout() error {
	return ClearCredentials()
}

// LoadOrRefresh carrega as credenciais do disco e renova se expiradas.
// Retorna nil se não houver credenciais (usuário não fez login).
func LoadOrRefresh(ctx context.Context) (*APIClient, error) {
	creds, err := LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("load credentials: %w", err)
	}
	if creds == nil {
		return nil, ErrNotAuthenticated
	}

	api := New(creds)

	// Se o token expirou, tentar refresh
	if creds.IsExpired() {
		if !creds.IsRefreshable() {
			return nil, ErrSessionExpired
		}
		if err := api.refreshToken(ctx); err != nil {
			return nil, fmt.Errorf("refresh expired token: %w", err)
		}
	}

	return api, nil
}

// Erros comuns de auth.
var (
	ErrNotAuthenticated = fmt.Errorf("not authenticated — run 'resma auth login' first")
	ErrSessionExpired   = fmt.Errorf("session expired and no refresh token — run 'resma auth login' again")
)

// parseRoleFromToken extrai o role do JWT sem validar a assinatura.
// Usado apenas para display (o API valida o token de verdade).
// Se falhar, retorna "unknown".
func parseRoleFromToken(tokenStr string) string {
	// JWT formato: header.payload.signature
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return "unknown"
	}
	// Decodificar payload (base64url sem padding)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "unknown"
	}
	var claims struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "unknown"
	}
	if claims.Role == "" {
		return "unknown"
	}
	return claims.Role
}
