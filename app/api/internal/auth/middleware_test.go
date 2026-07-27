package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper para criar request com UserContext no contexto
func reqWithUser(user *UserContext) *http.Request {
	r := httptest.NewRequest("GET", "/test", nil)
	if user != nil {
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, user))
	}
	return r
}

func TestRequireRole_AllowedRole(t *testing.T) {
	called := false
	handler := RequireRole(RoleOwner, RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	user := &UserContext{Sub: "1", Username: "admin", Role: "admin"}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reqWithUser(user))

	if !called {
		t.Error("handler was not called for allowed role")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireRole_InsufficientRole(t *testing.T) {
	called := false
	handler := RequireRole(RoleOwner, RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	user := &UserContext{Sub: "2", Username: "viewer", Role: "user"}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reqWithUser(user))

	if called {
		t.Error("handler was called for insufficient role (should be blocked)")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (403)", w.Code, http.StatusForbidden)
	}
}

func TestRequireRole_NoUserInContext(t *testing.T) {
	called := false
	handler := RequireRole(RoleOwner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reqWithUser(nil))

	if called {
		t.Error("handler was called without user in context (should be blocked)")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (401)", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole_OwnerOnly(t *testing.T) {
	called := false
	handler := RequireRole(RoleOwner)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	// admin não deve passar em RequireRole(RoleOwner)
	user := &UserContext{Sub: "3", Username: "admin", Role: "admin"}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, reqWithUser(user))

	if called {
		t.Error("admin passed RequireRole(RoleOwner) — should be blocked")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (403)", w.Code, http.StatusForbidden)
	}
}

func TestUserFromContext(t *testing.T) {
	// Sem user no contexto
	r := httptest.NewRequest("GET", "/", nil)
	if u := UserFromContext(r.Context()); u != nil {
		t.Error("expected nil user from empty context")
	}

	// Com user no contexto
	user := &UserContext{Sub: "1", Username: "test", Role: "admin"}
	ctx := context.WithValue(r.Context(), ctxKeyUser, user)
	if u := UserFromContext(ctx); u == nil || u.Username != "test" {
		t.Error("expected user from context")
	}
}
