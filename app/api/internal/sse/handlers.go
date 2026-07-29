// Package sse — handlers HTTP para endpoints SSE e sessão cookie.
//
// Endpoints:
//   - POST /api/sse/session — troca JWT bearer por cookie sse_session HttpOnly
//   - DELETE /api/sse/session — invalida cookie
//   - GET /api/sse/metrics — stream de métricas
//   - GET /api/sse/dashboard — stream de dashboard
//   - GET /api/sse/events — stream de eventos Docker
//   - GET /api/sse/services — stream de mudanças de serviços
//   - GET /api/sse/nodes — stream de mudanças de nós
//   - GET /api/sse/tasks — stream de mudanças de tasks (Fase 7)
//   - GET /api/sse/agents — stream de mudanças de agents (Fase 7)
package sse

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/auth"
)

// Handler coordena os endpoints SSE e o broker.
type Handler struct {
	broker *Broker
	auth   *auth.Service
	log    *slog.Logger

	mu       sync.RWMutex
	sessions map[string]sessionInfo // sse_session cookie value -> info
}

type sessionInfo struct {
	UserID   string
	Username string
	Expires  time.Time
}

// NewHandler cria um novo Handler SSE.
func NewHandler(broker *Broker, authSvc *auth.Service) *Handler {
	return &Handler{
		broker:   broker,
		auth:     authSvc,
		log:      slog.Default().With("component", "sse-handler"),
		sessions: make(map[string]sessionInfo),
	}
}

// Publish publica um evento em um tópico via broker. Conveniência para
// handlers e collector publicarem eventos SSE sem acessar o broker direto.
func (h *Handler) Publish(topic, eventType string, payload any) {
	h.broker.Publish(topic, Event{Type: eventType, Payload: payload})
}

// SubscriberCount retorna o número de subscribers ativos para um tópico.
// Usado pelo collector para saber se vale a pena construir e publicar o
// payload do ServiceDetail (só publica se alguém estiver vendo a página).
func (h *Handler) SubscriberCount(topic string) int {
	return h.broker.SubscriberCount(topic)
}

// SubscribedTopicsByPrefix retorna os tópicos ativos que começam com o prefixo.
// Usado pelo collector para descobrir quais tópicos dinâmicos
// "container-detail/{id}" têm subscribers ativos sem iterar sobre todos os
// containers conhecidos.
func (h *Handler) SubscribedTopicsByPrefix(prefix string) []string {
	return h.broker.SubscribedTopicsByPrefix(prefix)
}

// RegisterRoutes registra todos os endpoints SSE no mux.
// O mux deve ser o mux principal (não o interno com JWTMiddleware).
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Sessão cookie (requer JWT bearer para criar)
	mux.HandleFunc("POST /api/sse/session", h.handleCreateSession)
	mux.HandleFunc("DELETE /api/sse/session", h.handleDeleteSession)

	// Endpoints SSE (validam cookie OU Authorization)
	mux.HandleFunc("GET /api/sse/metrics", h.handleSSE("metrics"))
	mux.HandleFunc("GET /api/sse/dashboard", h.handleSSE("dashboard"))
	mux.HandleFunc("GET /api/sse/events", h.handleSSE("events"))
	mux.HandleFunc("GET /api/sse/services", h.handleSSE("services"))
	mux.HandleFunc("GET /api/sse/nodes", h.handleSSE("nodes"))
	// Fase 7 — multi-node
	mux.HandleFunc("GET /api/sse/tasks", h.handleSSE("tasks"))
	mux.HandleFunc("GET /api/sse/agents", h.handleSSE("agents"))
	// Tópico change-log — disparado pelo scheduler quando um schedule executa
	mux.HandleFunc("GET /api/sse/change-log", h.handleSSE("change-log"))
	// Tópico service-detail/{name} — disparado pelo collector a cada coleta
	// para cada serviço com subscriber ativo. Payload = stats+metrics+containers+tasks+health.
	mux.HandleFunc("GET /api/sse/service-detail/{name}", h.handleSSEServiceDetail)
	// Tópico container-detail/{id} — disparado pelo collector a cada coleta
	// para cada container com subscriber ativo. Payload = stats+metrics+network.
	mux.HandleFunc("GET /api/sse/container-detail/{id}", h.handleSSEContainerDetail)
}

// handleCreateSession troca um JWT bearer por um cookie sse_session HttpOnly.
// O frontend chama este endpoint com o JWT no header Authorization antes de
// abrir a conexão SSE (EventSource não suporta headers custom).
func (h *Handler) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	// Validar JWT bearer
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(w, http.StatusUnauthorized, "missing or invalid Authorization header")
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := h.auth.ValidateAccessToken(r.Context(), tokenStr)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	// Gerar session ID aleatório
	sessionBytes := make([]byte, 32)
	if _, err := rand.Read(sessionBytes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to generate session")
		return
	}
	sessionID := hex.EncodeToString(sessionBytes)

	// Armazenar sessão (TTL 10 minutos)
	expires := time.Now().Add(10 * time.Minute)
	h.mu.Lock()
	h.sessions[sessionID] = sessionInfo{
		UserID:   claims.Sub,
		Username: claims.Username,
		Expires:  expires,
	}
	h.mu.Unlock()

	// Set cookie HttpOnly
	cookie := &http.Cookie{
		Name:     "sse_session",
		Value:    sessionID,
		MaxAge:   600, // 10 minutos
		Path:     "/api/sse/",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   r.TLS != nil, // Secure em HTTPS
	}
	http.SetCookie(w, cookie)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"expires": expires.Format(time.RFC3339),
		"message": "SSE session created",
	})
}

// handleDeleteSession invalida o cookie sse_session.
func (h *Handler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("sse_session")
	if err == nil {
		h.mu.Lock()
		delete(h.sessions, cookie.Value)
		h.mu.Unlock()
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "sse_session",
		Value:    "",
		MaxAge:   -1,
		Path:     "/api/sse/",
		HttpOnly: true,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "SSE session deleted"})
}

// handleSSE retorna um handler para um tópico SSE específico.
// Valida auth via cookie sse_session OU header Authorization.
func (h *Handler) handleSSE(topic string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.validateSSEAuth(r) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		h.broker.ServeHTTP(w, r, topic)
	}
}

// handleSSEServiceDetail registra um subscriber no tópico "service-detail/{name}".
// O collector publica o payload completo (stats+metrics+containers+tasks+health)
// neste tópico a cada coleta, mas apenas se houver subscribers ativos.
func (h *Handler) handleSSEServiceDetail(w http.ResponseWriter, r *http.Request) {
	if !h.validateSSEAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	name := r.PathValue("name")
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "missing service name")
		return
	}
	topic := "service-detail/" + name
	h.broker.ServeHTTP(w, r, topic)
}

// handleSSEContainerDetail registra um subscriber no tópico "container-detail/{id}".
// O collector publica o payload completo (stats+metrics+network) neste tópico
// a cada coleta, mas apenas se houver subscribers ativos.
func (h *Handler) handleSSEContainerDetail(w http.ResponseWriter, r *http.Request) {
	if !h.validateSSEAuth(r) {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing container id")
		return
	}
	topic := "container-detail/" + id
	h.broker.ServeHTTP(w, r, topic)
}

// validateSSEAuth valida cookie sse_session OU header Authorization.
func (h *Handler) validateSSEAuth(r *http.Request) bool {
	// Tentar cookie primeiro
	if cookie, err := r.Cookie("sse_session"); err == nil {
		h.mu.RLock()
		info, ok := h.sessions[cookie.Value]
		h.mu.RUnlock()
		if ok && time.Now().Before(info.Expires) {
			return true
		}
		// Sessão expirada — remover
		if ok {
			h.mu.Lock()
			delete(h.sessions, cookie.Value)
			h.mu.Unlock()
		}
	}

	// Tentar Authorization bearer
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := h.auth.ValidateAccessToken(r.Context(), tokenStr); err == nil {
			return true
		}
	}

	return false
}

// CleanupExpired remove sessões expiradas periodicamente.
func (h *Handler) CleanupExpired() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for id, info := range h.sessions {
		if now.After(info.Expires) {
			delete(h.sessions, id)
		}
	}
}

// StartCleanup inicia goroutine de cleanup periódico.
func (h *Handler) StartCleanup(ctx interface{ Done() <-chan struct{} }) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.CleanupExpired()
			}
		}
	}()
}

// writeJSONError escreve um erro JSON.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
