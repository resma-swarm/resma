// Package server — /api/auth/* handlers.
//
// Portado de backend/routers/auth.py. 4 endpoints sem auth (status,
// onboarding, login, refresh) + 3 com auth (logout, me, change-password).
// + API key CRUD (novo, 0b.4).
package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/resma/api/internal/auth"
)

// registerPublicAuthRoutes registra as rotas de auth SEM JWT (onboarding flow).
// Estas rotas são registradas no mux principal sem JWTMiddleware.
func (s *Server) registerPublicAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/status", s.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/onboarding", s.handleOnboarding)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
}

// registerAuthRoutes registra as rotas de auth COM JWT no mux interno.
func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	// Com auth (já protegido pelo JWTMiddleware no mux pai)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)
	mux.HandleFunc("POST /api/auth/change-password", s.handleChangePassword)
	mux.HandleFunc("PUT /api/auth/profile", s.handleUpdateProfile)
}

// handleAuthStatus retorna se o sistema já tem usuário admin criado.
func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	count, err := s.db.GetUserCount(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check user count")
		return
	}
	writeOK(w, map[string]bool{"initialized": count > 0})
}

// handleOnboarding cria o primeiro usuário admin.
func (s *Server) handleOnboarding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	result, err := s.auth.DoOnboarding(r.Context(), req.Username, req.Password)
	if err != nil {
		if err == auth.ErrOnboardingCompleted {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// handleLogin autentica um usuário e retorna tokens.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	result, err := s.auth.DoLogin(r.Context(), req.Username, req.Password, ip)
	if err != nil {
		writeError(w, auth.ErrorToStatus(err), err.Error())
		return
	}
	writeOK(w, result)
}

// handleRefresh emite um novo access token a partir de um refresh token.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token required")
		return
	}
	result, err := s.auth.DoRefresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, auth.ErrorToStatus(err), err.Error())
		return
	}
	writeOK(w, result)
}

// handleLogout revoga o refresh token.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	_ = s.auth.DoLogout(r.Context(), req.RefreshToken)
	writeOK(w, map[string]string{"message": "Logged out"})
}

// handleMe retorna info do usuário autenticado.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	// Buscar do DB para incluir name (display name) sempre atualizado
	uid, _ := strconv.Atoi(user.Sub)
	dbUser, err := s.db.GetUserByID(r.Context(), int32(uid))
	if err != nil || dbUser == nil {
		// Fallback para JWT claims se DB falhar
		writeOK(w, map[string]string{
			"id":       user.Sub,
			"username": user.Username,
			"role":     user.Role,
			"name":     "",
		})
		return
	}
	writeOK(w, map[string]string{
		"id":       user.Sub,
		"username": dbUser.Username,
		"role":     dbUser.Role,
		"name":     dbUser.Name,
	})
}

// handleChangePassword troca a senha do usuário autenticado.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "current_password and new_password required")
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	userID, err := strconv.Atoi(user.Sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid user id")
		return
	}
	if err := s.auth.DoChangePassword(r.Context(), int32(userID), req.CurrentPassword, req.NewPassword); err != nil {
		writeError(w, auth.ErrorToStatus(err), err.Error())
		return
	}
	writeOK(w, map[string]string{"message": "Password changed. Please login again."})
}

// handleUpdateProfile atualiza o perfil do usuário autenticado (atualmente: name).
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	userID, err := strconv.Atoi(user.Sub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid user id")
		return
	}
	if err := s.db.UpdateUserName(r.Context(), int32(userID), strings.TrimSpace(req.Name)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]any{"name": strings.TrimSpace(req.Name)})
}
