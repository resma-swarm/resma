// Package auth — HTTP middleware para JWT e API key.
//
// Dois middlewares distintos:
//   - JWTMiddleware: valida JWT access token no header Authorization: Bearer.
//     Usado nas rotas internas /api/* (frontend).
//   - APIKeyMiddleware: valida API key no header Authorization: Bearer
//     resma_key_... ou X-API-Key. Usado nas rotas públicas /api/v1/*.
//
// Ambos populam o request context com o usuário/key autenticado.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/resma-swarm/resma/app/api/internal/db"
)

// contextKey é um tipo privado para keys de contexto.
type contextKey int

const (
	ctxKeyUser   contextKey = iota // *UserContext
	ctxKeyAPIKey                   // *db.APIKey
)

// UserFromContext extrai o UserContext do request context.
func UserFromContext(ctx context.Context) *UserContext {
	v, _ := ctx.Value(ctxKeyUser).(*UserContext)
	return v
}

// APIKeyFromContext extrai a APIKey do request context.
func APIKeyFromContext(ctx context.Context) *db.APIKey {
	v, _ := ctx.Value(ctxKeyAPIKey).(*db.APIKey)
	return v
}

// JWTMiddleware valida um JWT access token e popula o contexto com UserContext.
// Rejeita requests sem token válido com 401.
func (s *Service) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeAuthError(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}
		user, err := s.ValidateAccessToken(r.Context(), token)
		if err != nil {
			writeAuthError(w, err.Error(), ErrorToStatus(err))
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// APIKeyMiddleware valida uma API key e popula o contexto com *db.APIKey.
// Rejeita requests sem key válida com 401.
func (s *Service) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractAPIKey(r)
		if key == "" {
			writeAuthError(w, "missing API key", http.StatusUnauthorized)
			return
		}
		apiKey, err := s.ValidateAPIKey(r.Context(), key)
		if err != nil {
			writeAuthError(w, err.Error(), ErrorToStatus(err))
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyAPIKey, apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireScope é um middleware que verifica se a API key no contexto tem
// o scope necessário. Deve ser usado após APIKeyMiddleware.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := APIKeyFromContext(r.Context())
			if apiKey == nil {
				writeAuthError(w, "api key not in context", http.StatusUnauthorized)
				return
			}
			if !HasScope(apiKey, scope) {
				writeAuthError(w, "insufficient scope", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole é um middleware que verifica se o usuário no contexto tem
// um dos roles permitidos. Deve ser usado após JWTMiddleware.
// Retorna 401 se o usuário não estiver no contexto, 403 se o role for insuficiente.
func RequireRole(allowedRoles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				writeAuthError(w, "user not in context", http.StatusUnauthorized)
				return
			}
			userRole := Role(user.Role)
			for _, allowed := range allowedRoles {
				if userRole == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeAuthError(w, "insufficient role", http.StatusForbidden)
		})
	}
}

// extractBearerToken extrai o token do header "Authorization: Bearer <token>".
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// extractAPIKey extrai a API key de "Authorization: Bearer resma_key_..." ou
// "X-API-Key: resma_key_...".
func extractAPIKey(r *http.Request) string {
	// Tentar X-API-Key primeiro
	if key := r.Header.Get("X-API-Key"); key != "" {
		return strings.TrimSpace(key)
	}
	// Tentar Authorization: Bearer
	token := extractBearerToken(r)
	if strings.HasPrefix(token, APIKeyPrefix) {
		return token
	}
	return ""
}

// writeAuthError escreve uma resposta de erro de auth em JSON.
func writeAuthError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
