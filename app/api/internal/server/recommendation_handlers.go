// Package server — /api/recommendations/* handlers.
//
// Portado de backend/routers/recommendations.py. A lógica ML (numpy,
// scikit-learn) está no ML sidecar Python (0b.8). Os handlers chamam
// o ML sidecar via mlclient com fallback graceful.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/resma/api/internal/auth"
	"github.com/resma/api/internal/docker"
)

// registerRecommendationRoutes registra as rotas de recommendations.
// Fase 8: POST /apply exige owner/admin (modifica recursos do Swarm).
// POST /recalculate é permitido para todos (apenas dispara ML, não modifica).
func (s *Server) registerRecommendationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/recommendations", s.handleListRecommendations)
	mux.HandleFunc("GET /api/recommendations/triggers", s.handleGetTriggers)
	mux.HandleFunc("GET /api/recommendations/storage", s.handleStorageRecommendations)
	mux.HandleFunc("POST /api/recommendations/recalculate", s.handleRecalculate)
	mux.HandleFunc("GET /api/recommendations/{service}", s.handleGetRecommendation)
	mux.HandleFunc("POST /api/recommendations/{service}/recalculate", s.handleRecalculateService)

	// Rotas de escrita (apply) — owner/admin apenas
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("POST /api/recommendations/{service}/apply", rbac(http.HandlerFunc(s.handleApplyRecommendation)))
}

// handleListRecommendations lista recomendações de todos os serviços.
func (s *Server) handleListRecommendations(w http.ResponseWriter, r *http.Request) {
	result, err := s.buildRecommendations(r.Context())
	if err != nil || result == nil {
		writeOK(w, []any{})
		return
	}
	writeOK(w, result)
}

// handleGetTriggers retorna triggers de recomendação.
func (s *Server) handleGetTriggers(w http.ResponseWriter, r *http.Request) {
	if result, err := s.ml.EvaluateTriggers(r.Context()); err == nil {
		writeOK(w, result)
		return
	}
	writeOK(w, []any{})
}

// handleStorageRecommendations retorna recomendações de storage.
func (s *Server) handleStorageRecommendations(w http.ResponseWriter, r *http.Request) {
	if result, err := s.ml.AnalyzeStorage(r.Context()); err == nil {
		writeOK(w, result)
		return
	}
	writeOK(w, map[string]any{
		"summary":         map[string]any{},
		"recommendations": []any{},
	})
}

// handleRecalculate recalcula recomendações de todos os serviços.
func (s *Server) handleRecalculate(w http.ResponseWriter, r *http.Request) {
	if result, err := s.ml.AnalyzeAll(r.Context()); err == nil {
		if arr, ok := result.([]any); ok {
			writeOK(w, map[string]any{
				"recalculated": len(arr),
				"results":      result,
			})
			return
		}
	}
	writeOK(w, map[string]any{"recalculated": 0, "results": []any{}})
}

// handleGetRecommendation retorna recomendação de um serviço específico.
func (s *Server) handleGetRecommendation(w http.ResponseWriter, r *http.Request) {
	service := pathValue(r, "service")
	if err := s.validateService(r, service); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if result, err := s.ml.AnalyzeService(r.Context(), service); err == nil {
		writeOK(w, result)
		return
	}
	writeOK(w, map[string]any{
		"service":   service,
		"samples":   0,
		"status":    "collecting_data",
		"suggested": map[string]any{},
	})
}

// handleRecalculateService recalcula recomendação de um serviço.
func (s *Server) handleRecalculateService(w http.ResponseWriter, r *http.Request) {
	service := pathValue(r, "service")
	if err := s.validateService(r, service); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if result, err := s.ml.AnalyzeService(r.Context(), service); err == nil {
		writeOK(w, result)
		return
	}
	writeOK(w, map[string]any{
		"service":   service,
		"samples":   0,
		"status":    "collecting_data",
		"suggested": map[string]any{},
	})
}

// handleApplyRecommendation aplica recomendação de recursos a um serviço.
func (s *Server) handleApplyRecommendation(w http.ResponseWriter, r *http.Request) {
	service := pathValue(r, "service")
	if err := s.validateService(r, service); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		CPULimit       *float64 `json:"cpu_limit"`
		MemLimit       *int64   `json:"mem_limit"`
		CPUReservation *float64 `json:"cpu_reservation"`
		MemReservation *int64   `json:"mem_reservation"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	ctx := r.Context()

	// UpdateServiceResources: cpuLimit, memLimit *float64, cpuReservation, memReservation *int64
	var memLimitFloat *float64
	if req.MemLimit != nil {
		mlf := float64(*req.MemLimit)
		memLimitFloat = &mlf
	}
	var cpuResInt *int64
	if req.CPUReservation != nil {
		cri := int64(*req.CPUReservation * 1e9)
		cpuResInt = &cri
	}

	result, err := s.docker.UpdateServiceResources(ctx, service,
		req.CPULimit, memLimitFloat, cpuResInt, req.MemReservation)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if result.Success {
		writeOK(w, map[string]any{
			"success": true,
			"message": fmt.Sprintf("Resources applied to '%s'", service),
		})
		return
	}

	writeOK(w, map[string]any{
		"success": false,
		"message": fmt.Sprintf("Failed to apply resources to '%s': %s", service, result.Error),
	})
}

// validateService verifica se um serviço existe, não está arquivado e está online.
func (s *Server) validateService(r *http.Request, service string) error {
	ctx := r.Context()
	statusMap, err := s.docker.GetServiceStatusMap(ctx)
	if err != nil {
		return fmt.Errorf("docker indisponível")
	}
	if _, ok := statusMap[service]; !ok {
		return fmt.Errorf("'%s' não é um service do Swarm", service)
	}
	registry, _ := s.db.GetServiceRegistry(ctx)
	if reg, ok := registry[service]; ok && reg.Status == "archived" {
		return fmt.Errorf("'%s' está arquivado", service)
	}
	if statusMap[service] == "offline" {
		return fmt.Errorf("'%s' está offline", service)
	}
	return nil
}

// logChangeLog é um helper para registrar mudanças no change_log.
// TODO 0b.5: implementar com db.AddChangeLog quando ApplyRequest for finalizado
func (s *Server) logChangeLog(ctx context.Context, service, action, source string,
	current docker.ServiceResources, cpuLimitAfter, memLimitAfter *float64,
	cpuResAfter, memResAfter *int64, user, status, errMsg, dockerResp string) {
	// TODO: implementar com db.AddChangeLog
}
