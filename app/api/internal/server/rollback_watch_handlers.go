// Package server — /api/rollback-watches/* handlers (Right-Sizing Studio R5).
//
// Endpoints admin para listar, detalhar, rollback manual e cancelar watches.
// Leitura: qualquer role autenticada. Mutação: owner/admin apenas.
package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/resma/api/internal/auth"
	"github.com/resma/api/internal/db"
)

// registerRollbackWatchRoutes registra as rotas de rollback watches.
func (s *Server) registerRollbackWatchRoutes(mux *http.ServeMux) {
	// Leitura — qualquer role autenticada
	mux.HandleFunc("GET /api/rollback-watches", s.handleListRollbackWatches)
	mux.HandleFunc("GET /api/rollback-watches/{id}", s.handleGetRollbackWatch)

	// Mutação — owner/admin apenas
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("POST /api/rollback-watches/{id}/rollback", rbac(http.HandlerFunc(s.handleManualRollback)))
	mux.Handle("POST /api/rollback-watches/{id}/cancel", rbac(http.HandlerFunc(s.handleCancelWatch)))
}

// watchToJSON converte db.RollbackWatch para o formato JSON da API.
func watchToJSON(w db.RollbackWatch) map[string]any {
	m := map[string]any{
		"id":                       w.ID,
		"change_log_id":            w.ChangeLogID,
		"service":                  w.Service,
		"strategy":                 w.Strategy,
		"observation_window_hours": w.ObservationWindow,
		"criteria":                 json.RawMessage(w.Criteria),
		"status":                   w.Status,
		"triggered_criteria":       nil,
		"started_at":               w.StartedAt.Format(time.RFC3339),
		"expires_at":               w.ExpiresAt.Format(time.RFC3339),
		"rolled_back_at":           nil,
		"cpu_limit_before":         nullFloatToAny(w.CPULimitBefore),
		"mem_limit_before":         nullInt64ToAny(w.MemLimitBefore),
		"cpu_reservation_before":   nullFloatToAny(w.CPUReservationBefore),
		"mem_reservation_before":   nullInt64ToAny(w.MemReservationBefore),
		"cpu_limit_after":          nullFloatToAny(w.CPULimitAfter),
		"mem_limit_after":          nullInt64ToAny(w.MemLimitAfter),
		"cpu_reservation_after":    nullFloatToAny(w.CPUReservationAfter),
		"mem_reservation_after":    nullInt64ToAny(w.MemReservationAfter),
	}
	if w.TriggeredCriteria.Valid {
		m["triggered_criteria"] = w.TriggeredCriteria.String
	}
	if w.RolledBackAt.Valid {
		m["rolled_back_at"] = w.RolledBackAt.Time.Format(time.RFC3339)
	}
	return m
}

func nullFloatToAny(n sql.NullFloat64) any {
	if !n.Valid {
		return nil
	}
	return n.Float64
}

func nullInt64ToAny(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

// handleListRollbackWatches lista watches com filtros opcionais.
func (s *Server) handleListRollbackWatches(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	service := q.Get("service")
	limit := int32(100)
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = int32(n)
		}
	}
	watches, err := s.db.ListRollbackWatches(r.Context(), status, service, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(watches))
	for _, watch := range watches {
		out = append(out, watchToJSON(watch))
	}
	writeOK(w, map[string]any{
		"watches": out,
		"total":   len(out),
	})
}

// handleGetRollbackWatch retorna detalhes de um watch específico.
func (s *Server) handleGetRollbackWatch(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch id")
		return
	}
	watch, err := s.db.GetRollbackWatchByID(r.Context(), int32(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if watch == nil {
		writeError(w, http.StatusNotFound, "watch not found")
		return
	}
	writeOK(w, watchToJSON(*watch))
}

// handleManualRollback executa rollback manual para os valores before.
func (s *Server) handleManualRollback(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch id")
		return
	}
	ctx := r.Context()
	watch, err := s.db.GetRollbackWatchByID(ctx, int32(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if watch == nil {
		writeError(w, http.StatusNotFound, "watch not found")
		return
	}
	if watch.Status != "monitoring" {
		writeError(w, http.StatusConflict, fmt.Sprintf("watch is not monitoring (status=%s)", watch.Status))
		return
	}

	// Reverter via Docker (valores before)
	var cpuLimit *float64
	if watch.CPULimitBefore.Valid {
		v := watch.CPULimitBefore.Float64
		cpuLimit = &v
	}
	var memLimit *float64
	if watch.MemLimitBefore.Valid {
		v := float64(watch.MemLimitBefore.Int64)
		memLimit = &v
	}
	var cpuRes *int64
	if watch.CPUReservationBefore.Valid {
		v := int64(watch.CPUReservationBefore.Float64 * 1e9)
		cpuRes = &v
	}
	var memRes *int64
	if watch.MemReservationBefore.Valid {
		v := watch.MemReservationBefore.Int64
		memRes = &v
	}

	result, err := s.docker.UpdateServiceResources(ctx, watch.Service,
		cpuLimit, memLimit, cpuRes, memRes)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("docker rollback failed: %v", err))
		return
	}

	// Registrar no change_log (action=rollback, source=manual)
	username := ""
	if user := auth.UserFromContext(ctx); user != nil {
		username = user.Username
	}
	status := "completed"
	if !result.Success {
		status = "failed"
	}
	changeLogID, _ := s.db.AddChangeLog(ctx, db.ChangeLogEntry{
		Service:              watch.Service,
		Action:               "rollback",
		Source:               "manual",
		CPULimitBefore:       watch.CPULimitAfter,
		MemLimitBefore:       watch.MemLimitAfter,
		CPUReservationBefore: watch.CPUReservationAfter,
		MemReservationBefore: watch.MemReservationAfter,
		CPULimitAfter:        watch.CPULimitBefore,
		MemLimitAfter:        watch.MemLimitBefore,
		CPUReservationAfter:  watch.CPUReservationBefore,
		MemReservationAfter:  watch.MemReservationBefore,
		User:                 nullStrPtr(username),
		Status:               status,
		DockerResponse:       nullStrPtr(result.Error),
	})

	// Atualizar watch
	now := time.Now()
	_ = s.db.UpdateRollbackWatchStatus(ctx, watch.ID, "rolled_back", "manual", &now)

	// SSE
	if entries, err := s.buildChangeLog(ctx, "", 100); err == nil && entries != nil {
		s.sse.Publish("change-log", "change-log", entries)
	}
	s.sse.Publish("events", "rollback_manual", map[string]any{
		"service": watch.Service,
		"user":    username,
		"time":    now.Format(time.RFC3339),
	})

	writeOK(w, map[string]any{
		"success":       result.Success,
		"message":       fmt.Sprintf("Rollback manual executado para %s", watch.Service),
		"change_log_id": changeLogID,
		"watch_id":      watch.ID,
		"reverted_to": map[string]any{
			"cpu_limit":       nullFloatToAny(watch.CPULimitBefore),
			"mem_limit":       nullInt64ToAny(watch.MemLimitBefore),
			"cpu_reservation": nullFloatToAny(watch.CPUReservationBefore),
			"mem_reservation": nullInt64ToAny(watch.MemReservationBefore),
		},
	})
}

// handleCancelWatch cancela um watch (para de monitorar, não reverte).
func (s *Server) handleCancelWatch(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid watch id")
		return
	}
	ctx := r.Context()
	watch, err := s.db.GetRollbackWatchByID(ctx, int32(id))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if watch == nil {
		writeError(w, http.StatusNotFound, "watch not found")
		return
	}
	if watch.Status != "monitoring" {
		writeError(w, http.StatusConflict, fmt.Sprintf("watch is not monitoring (status=%s)", watch.Status))
		return
	}

	_ = s.db.UpdateRollbackWatchStatus(ctx, watch.ID, "cancelled", "", nil)

	username := ""
	if user := auth.UserFromContext(ctx); user != nil {
		username = user.Username
	}
	s.sse.Publish("events", "watch_cancelled", map[string]any{
		"service": watch.Service,
		"user":    username,
		"time":    time.Now().Format(time.RFC3339),
	})

	writeOK(w, map[string]any{
		"success":  true,
		"message":  fmt.Sprintf("Watch cancelado — monitoramento interrompido para %s", watch.Service),
		"watch_id": watch.ID,
		"status":   "cancelled",
	})
}
