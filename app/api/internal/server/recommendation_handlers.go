// Package server — /api/recommendations/* handlers.
//
// Portado de backend/routers/recommendations.py. A lógica ML (numpy,
// scikit-learn) está no ML sidecar Python (0b.8). Os handlers chamam
// o ML sidecar via mlclient com fallback graceful.
package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/resma-swarm/resma/app/api/internal/auth"
	"github.com/resma-swarm/resma/app/api/internal/db"
	"github.com/resma-swarm/resma/app/api/internal/rollback"
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
// Right-Sizing Studio R5: estendido com change_log + rollback_watches.
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
		Rollback       *struct {
			Enabled                bool   `json:"enabled"`
			Strategy               string `json:"strategy"`
			ObservationWindowHours *int   `json:"observation_window_hours"`
			Criteria               *struct {
				OOM         bool `json:"oom"`
				Throttle    bool `json:"throttle"`
				MemPressure bool `json:"mem_pressure"`
			} `json:"criteria"`
		} `json:"rollback"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	ctx := r.Context()

	// 1. Obter valores atuais (snapshot de rollback)
	current, _ := s.docker.GetServiceResources(ctx, service)

	// 2. Aplicar via Docker
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

	// 3. Obter username do contexto (para change_log)
	username := ""
	if user := auth.UserFromContext(ctx); user != nil {
		username = user.Username
	}

	// 4. Registrar no change_log (action=apply, source=manual)
	changeLogID, _ := s.db.AddChangeLog(ctx, db.ChangeLogEntry{
		Service:              service,
		Action:               "apply",
		Source:               "manual",
		CPULimitBefore:       nullFloat(current.CPULimit),
		MemLimitBefore:       nullInt64(current.MemLimit),
		CPUReservationBefore: nullFloat(current.CPUReservation),
		MemReservationBefore: nullInt64(current.MemReservation),
		CPULimitAfter:        nullFloatPtr(req.CPULimit),
		MemLimitAfter:        nullInt64Ptr(req.MemLimit),
		CPUReservationAfter:  nullFloatPtr(req.CPUReservation),
		MemReservationAfter:  nullInt64Ptr(req.MemReservation),
		User:                 nullStrPtr(username),
		Status:               boolStr(result.Success, "completed", "failed"),
		DockerResponse:       nullStrPtr(result.Error),
	})

	// 5. Criar watch de rollback se habilitado (opt-in via config OU request body)
	rollbackEnabled := s.cfg.RollbackEnabled
	if req.Rollback != nil && req.Rollback.Enabled {
		rollbackEnabled = true
	}
	var watchID *int32
	if rollbackEnabled && changeLogID > 0 && result.Success {
		strategy := "deferred"
		if req.Rollback != nil && req.Rollback.Strategy != "" {
			strategy = req.Rollback.Strategy
		}
		window := s.cfg.RollbackDefaultWindow
		if req.Rollback != nil && req.Rollback.ObservationWindowHours != nil {
			window = *req.Rollback.ObservationWindowHours
		}
		if window < 1 {
			window = 1
		}
		if window > 168 {
			window = 168
		}
		criteria := rollback.Criteria{OOM: true, Throttle: true}
		if req.Rollback != nil && req.Rollback.Criteria != nil {
			criteria = rollback.Criteria{
				OOM:         req.Rollback.Criteria.OOM,
				Throttle:    req.Rollback.Criteria.Throttle,
				MemPressure: req.Rollback.Criteria.MemPressure,
			}
		}
		criteriaJSON, _ := json.Marshal(criteria)
		id, err := s.db.CreateRollbackWatch(ctx, db.RollbackWatch{
			ChangeLogID:          changeLogID,
			Service:              service,
			CPULimitBefore:       nullFloat(current.CPULimit),
			MemLimitBefore:       nullInt64(current.MemLimit),
			CPUReservationBefore: nullFloat(current.CPUReservation),
			MemReservationBefore: nullInt64(current.MemReservation),
			CPULimitAfter:        nullFloatPtr(req.CPULimit),
			MemLimitAfter:        nullInt64Ptr(req.MemLimit),
			CPUReservationAfter:  nullFloatPtr(req.CPUReservation),
			MemReservationAfter:  nullInt64Ptr(req.MemReservation),
			Strategy:             strategy,
			ObservationWindow:    window,
			Criteria:             string(criteriaJSON),
		})
		if err == nil {
			watchID = &id
		}
	}

	// 6. Publicar SSE change-log (frontend atualiza sem refetch)
	if entries, err := s.buildChangeLog(ctx, "", 100); err == nil && entries != nil {
		s.sse.Publish("change-log", "change-log", entries)
	}

	if result.Success {
		msg := fmt.Sprintf("Resources applied to '%s'", service)
		if watchID != nil {
			msg = fmt.Sprintf("Resources applied to '%s'. Rollback watch active.", service)
		}
		writeOK(w, map[string]any{
			"success":           true,
			"message":           msg,
			"change_log_id":     changeLogID,
			"rollback_watch_id": watchID,
		})
		return
	}

	writeOK(w, map[string]any{
		"success":           false,
		"message":           fmt.Sprintf("Failed to apply resources to '%s': %s", service, result.Error),
		"change_log_id":     changeLogID,
		"rollback_watch_id": nil,
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

// --- helpers para sql.Null* (change_log + rollback_watches) ---

// nullFloat converte float64 em sql.NullFloat64 (valid se != 0).
func nullFloat(f float64) sql.NullFloat64 {
	if f == 0 {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: f, Valid: true}
}

// nullFloatPtr converte *float64 em sql.NullFloat64.
func nullFloatPtr(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

// nullInt64 converte int64 em sql.NullInt64 (valid if != 0).
func nullInt64(n int64) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}

// nullInt64Ptr converte *int64 em sql.NullInt64.
func nullInt64Ptr(n *int64) sql.NullInt64 {
	if n == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *n, Valid: true}
}

// nullStrPtr converte string em sql.NullString (valid if non-empty).
func nullStrPtr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// boolStr retorna trueVal se cond, senão falseVal.
func boolStr(cond bool, trueVal, falseVal string) string {
	if cond {
		return trueVal
	}
	return falseVal
}
