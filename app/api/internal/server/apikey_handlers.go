// Package server — /api/auth/api-keys/* handlers (CRUD).
//
// Novo em 0b.4 — não existe no Python. Endpoints internos para admin
// gerenciar API keys via UI.
package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/resma/api/internal/auth"
	"github.com/resma/api/internal/db"
)

// registerAPIKeyRoutes registra as rotas de API key CRUD.
// Fase 8: todas as operações de API keys exigem role owner ou admin.
func (s *Server) registerAPIKeyRoutes(mux *http.ServeMux) {
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("GET /api/auth/api-keys", rbac(http.HandlerFunc(s.handleListAPIKeys)))
	mux.Handle("POST /api/auth/api-keys", rbac(http.HandlerFunc(s.handleCreateAPIKey)))
	mux.Handle("DELETE /api/auth/api-keys/{id}", rbac(http.HandlerFunc(s.handleRevokeAPIKey)))
	mux.Handle("PATCH /api/auth/api-keys/{id}", rbac(http.HandlerFunc(s.handleUpdateAPIKey)))
}

// handleListAPIKeys lista todas as API keys (sem o hash).
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.db.ListAPIKeys(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		out = append(out, toAPIKeyResponse(k))
	}
	writeOK(w, out)
}

// apiKeyResponse é o DTO JSON-friendly para API keys (sql.NullTime → *string).
type apiKeyResponse struct {
	ID         int32   `json:"id"`
	Prefix     string  `json:"prefix"`
	Name       string  `json:"name"`
	Scopes     string  `json:"scopes"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at"`
	RevokedAt  *string `json:"revoked_at"`
}

func toAPIKeyResponse(k db.APIKey) apiKeyResponse {
	resp := apiKeyResponse{
		ID:        k.ID,
		Prefix:    k.KeyPrefix,
		Name:      k.Name,
		Scopes:    k.Scopes,
		CreatedAt: k.CreatedAt.Format(time.RFC3339),
	}
	if k.LastUsedAt.Valid {
		s := k.LastUsedAt.Time.Format(time.RFC3339)
		resp.LastUsedAt = &s
	}
	if k.RevokedAt.Valid {
		s := k.RevokedAt.Time.Format(time.RFC3339)
		resp.RevokedAt = &s
	}
	return resp
}

// handleCreateAPIKey cria uma nova API key e retorna o plaintext uma única vez.
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if req.Scopes == "" {
		req.Scopes = "read"
	}
	plaintext, id, err := s.auth.CreateAPIKey(r.Context(), req.Name, req.Scopes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":      id,
		"key":     plaintext,
		"name":    req.Name,
		"scopes":  req.Scopes,
		"message": "Save this key — it won't be shown again",
	})
}

// handleRevokeAPIKey revoga uma API key.
func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.db.RevokeAPIKey(r.Context(), int32(id)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"message": "API key revoked"})
}

// handleUpdateAPIKey atualiza o nome de uma API key.
func (s *Server) handleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := s.db.UpdateAPIKeyName(r.Context(), int32(id), req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]string{"message": "API key updated"})
}

// suppress unused import
var _ = auth.APIKeyPrefix
