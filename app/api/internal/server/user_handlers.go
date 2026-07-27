// Package server — /api/users/* handlers (Fase 8: User CRUD + RBAC).
//
// Endpoints para gestão de usuários (owner/admin apenas). Todas as mutações
// são auditadas em change_log (action='user_create'/'user_update'/'user_delete').
package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/resma/api/internal/auth"
	"github.com/resma/api/internal/db"
)

// registerUserRoutes registra as rotas de User CRUD.
// Fase 8: todas exigem role owner ou admin.
func (s *Server) registerUserRoutes(mux *http.ServeMux) {
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("GET /api/users", rbac(http.HandlerFunc(s.handleListUsers)))
	mux.Handle("POST /api/users", rbac(http.HandlerFunc(s.handleCreateUser)))
	mux.Handle("PATCH /api/users/{id}", rbac(http.HandlerFunc(s.handleUpdateUser)))
	mux.Handle("DELETE /api/users/{id}", rbac(http.HandlerFunc(s.handleDeleteUser)))
}

// handleListUsers lista todos os usuários (sem password_hash).
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if users == nil {
		users = []db.User{}
	}
	// Serializar sem password_hash (User.PasswordHash fica vazio pois ListUsers não o popula)
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{
			"id":       u.ID,
			"username": u.Username,
			"role":     u.Role,
			"name":     u.Name,
		})
	}
	writeOK(w, map[string]any{"users": out})
}

// handleCreateUser cria um novo usuário com role admin ou user.
// Role 'owner' é rejeitado (apenas 1 owner, criado via onboarding).
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Name     string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Name = strings.TrimSpace(req.Name)
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	// Validar role — apenas admin ou user (owner é único, via onboarding)
	if req.Role != string(auth.RoleAdmin) && req.Role != string(auth.RoleUser) {
		writeError(w, http.StatusBadRequest, "role must be 'admin' or 'user'")
		return
	}

	hash, err := s.auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	id, err := s.db.CreateUserWithRole(r.Context(), req.Username, hash, req.Role, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "username already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit em change_log (service='__users__', action='user_create')
	s.auditUserChange(r, "user_create", req.Username, req.Role, "")

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"username": req.Username,
		"role":     req.Role,
		"name":     req.Name,
	})
}

// handleUpdateUser atualiza o role de um usuário.
// Não permite promover para owner. Não permite rebaixar o owner atual.
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Role != string(auth.RoleAdmin) && req.Role != string(auth.RoleUser) {
		writeError(w, http.StatusBadRequest, "role must be 'admin' or 'user'")
		return
	}

	// Buscar usuário alvo
	target, err := s.db.GetUserByID(r.Context(), int32(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	// Proteger owner: não rebaixar
	if target.Role == string(auth.RoleOwner) {
		writeError(w, http.StatusForbidden, "cannot change owner role")
		return
	}

	if err := s.db.UpdateUserRole(r.Context(), int32(id), req.Role); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit
	s.auditUserChange(r, "user_update", target.Username, target.Role, req.Role)

	writeOK(w, map[string]any{"id": int32(id), "role": req.Role})
}

// handleDeleteUser remove um usuário. Não permite deletar o owner.
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Buscar usuário alvo
	target, err := s.db.GetUserByID(r.Context(), int32(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	// Proteger owner
	if target.Role == string(auth.RoleOwner) {
		writeError(w, http.StatusForbidden, "cannot delete owner")
		return
	}

	// Não permitir auto-deleção
	currentUser := auth.UserFromContext(r.Context())
	if currentUser != nil && currentUser.Sub == strconv.Itoa(id) {
		writeError(w, http.StatusForbidden, "cannot delete yourself")
		return
	}

	if err := s.db.DeleteUser(r.Context(), int32(id)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit
	s.auditUserChange(r, "user_delete", target.Username, target.Role, "")

	writeOK(w, map[string]string{"message": "user deleted"})
}

// auditUserChange registra uma mutação de usuário em change_log.
// service='__users__' é um sentinel para mudanças de governance (não de serviço).
func (s *Server) auditUserChange(r *http.Request, action, username, roleBefore, roleAfter string) {
	currentUser := auth.UserFromContext(r.Context())
	actor := "system"
	if currentUser != nil {
		actor = currentUser.Username
	}
	details := roleBefore
	if roleAfter != "" {
		details = roleBefore + " -> " + roleAfter
	}
	_, _ = s.db.AddChangeLog(r.Context(), db.ChangeLogEntry{
		Service:        "__users__",
		Action:         action,
		Source:         "ui",
		User:           toNullString(actor),
		Status:         "completed",
		DockerResponse: toNullString(details + " user=" + username),
	})
}

// toNullString converte string para sql.NullString (vira NULL se vazia).
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
