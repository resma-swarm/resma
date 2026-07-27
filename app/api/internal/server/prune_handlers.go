// Package server — /api/prune/* handlers (Fase 8: Data management).
//
// Endpoints para limpeza manual de dados stale/acumulados. Suporta dry-run
// (default) e real prune. Todas as operações de prune real são auditadas
// em change_log (action='prune_*').
//
// GET  /api/prune/preview          — contagem de stale/orphan/acumulado
// POST /api/prune/services-stale   — prune services stale
// POST /api/prune/nodes-stale      — prune nodes stale
// POST /api/prune/tasks-orphan     — prune tasks orphan
// POST /api/prune/metrics          — prune all metrics
// POST /api/prune/change-log       — prune all change_log
// POST /api/prune/volume-metrics   — prune all volume_metrics
package server

import (
	"net/http"
	"strconv"

	"github.com/resma/api/internal/auth"
	"github.com/resma/api/internal/db"
)

// registerPruneRoutes registra as rotas de prune.
// Fase 8: todas exigem role owner ou admin (operações destrutivas).
func (s *Server) registerPruneRoutes(mux *http.ServeMux) {
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("GET /api/prune/preview", rbac(http.HandlerFunc(s.handlePrunePreview)))
	mux.Handle("POST /api/prune/services-stale", rbac(http.HandlerFunc(s.handlePruneServicesStale)))
	mux.Handle("POST /api/prune/nodes-stale", rbac(http.HandlerFunc(s.handlePruneNodesStale)))
	mux.Handle("POST /api/prune/tasks-orphan", rbac(http.HandlerFunc(s.handlePruneTasksOrphan)))
	mux.Handle("POST /api/prune/metrics", rbac(http.HandlerFunc(s.handlePruneMetrics)))
	mux.Handle("POST /api/prune/change-log", rbac(http.HandlerFunc(s.handlePruneChangeLog)))
	mux.Handle("POST /api/prune/volume-metrics", rbac(http.HandlerFunc(s.handlePruneVolumeMetrics)))
}

// handlePrunePreview retorna contagens de dados stale/orphan/acumulados.
// Não modifica nada — usado pela UI para mostrar antes de confirmar.
func (s *Server) handlePrunePreview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	servicesStale, _ := s.db.CountStaleServices(ctx)
	nodesStale, _ := s.db.CountStaleNodes(ctx)
	tasksOrphan, _ := s.db.CountOrphanTasks(ctx)
	metrics, _ := s.db.CountMetrics(ctx)
	changeLog, _ := s.db.CountChangeLog(ctx)
	volumeMetrics, _ := s.db.CountVolumeMetrics(ctx)

	writeOK(w, map[string]any{
		"services_stale": servicesStale,
		"nodes_stale":    nodesStale,
		"tasks_orphan":   tasksOrphan,
		"metrics":        metrics,
		"change_log":     changeLog,
		"volume_metrics": volumeMetrics,
	})
}

// pruneRequest é o body comum a todos os endpoints de prune.
type pruneRequest struct {
	DryRun *bool `json:"dry_run"` // default true (segurança)
}

// isDryRun retorna true se dry_run não foi explicitamente setado como false.
func isDryRun(req *pruneRequest) bool {
	if req == nil || req.DryRun == nil {
		return true
	}
	return *req.DryRun
}

// handlePruneServicesStale remove services com status='stale'.
func (s *Server) handlePruneServicesStale(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	_ = decodeJSON(w, r, &req) // body opcional
	ctx := r.Context()

	count, _ := s.db.CountStaleServices(ctx)
	if isDryRun(&req) {
		writeOK(w, map[string]any{"dry_run": true, "would_delete": count, "deleted": 0})
		return
	}
	deleted, err := s.db.PruneStaleServices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditPrune(r, "prune_services_stale", deleted)
	writeOK(w, map[string]any{"dry_run": false, "would_delete": count, "deleted": deleted})
}

// handlePruneNodesStale remove nodes com status='stale'.
func (s *Server) handlePruneNodesStale(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	_ = decodeJSON(w, r, &req)
	ctx := r.Context()

	count, _ := s.db.CountStaleNodes(ctx)
	if isDryRun(&req) {
		writeOK(w, map[string]any{"dry_run": true, "would_delete": count, "deleted": 0})
		return
	}
	deleted, err := s.db.PruneStaleNodes(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditPrune(r, "prune_nodes_stale", deleted)
	writeOK(w, map[string]any{"dry_run": false, "would_delete": count, "deleted": deleted})
}

// handlePruneTasksOrphan remove tasks orphan/removed.
func (s *Server) handlePruneTasksOrphan(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	_ = decodeJSON(w, r, &req)
	ctx := r.Context()

	count, _ := s.db.CountOrphanTasks(ctx)
	if isDryRun(&req) {
		writeOK(w, map[string]any{"dry_run": true, "would_delete": count, "deleted": 0})
		return
	}
	deleted, err := s.db.PruneOrphanTasks(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditPrune(r, "prune_tasks_orphan", deleted)
	writeOK(w, map[string]any{"dry_run": false, "would_delete": count, "deleted": deleted})
}

// handlePruneMetrics remove todas as métricas (operação destrutiva).
func (s *Server) handlePruneMetrics(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	_ = decodeJSON(w, r, &req)
	ctx := r.Context()

	count, _ := s.db.CountMetrics(ctx)
	if isDryRun(&req) {
		writeOK(w, map[string]any{"dry_run": true, "would_delete": count, "deleted": 0})
		return
	}
	deleted, err := s.db.PruneMetrics(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditPrune(r, "prune_metrics", deleted)
	writeOK(w, map[string]any{"dry_run": false, "would_delete": count, "deleted": deleted})
}

// handlePruneChangeLog remove todo o change_log.
func (s *Server) handlePruneChangeLog(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	_ = decodeJSON(w, r, &req)
	ctx := r.Context()

	count, _ := s.db.CountChangeLog(ctx)
	if isDryRun(&req) {
		writeOK(w, map[string]any{"dry_run": true, "would_delete": count, "deleted": 0})
		return
	}
	deleted, err := s.db.PruneChangeLog(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditPrune(r, "prune_change_log", deleted)
	writeOK(w, map[string]any{"dry_run": false, "would_delete": count, "deleted": deleted})
}

// handlePruneVolumeMetrics remove todas as volume_metrics.
func (s *Server) handlePruneVolumeMetrics(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	_ = decodeJSON(w, r, &req)
	ctx := r.Context()

	count, _ := s.db.CountVolumeMetrics(ctx)
	if isDryRun(&req) {
		writeOK(w, map[string]any{"dry_run": true, "would_delete": count, "deleted": 0})
		return
	}
	deleted, err := s.db.PruneVolumeMetrics(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditPrune(r, "prune_volume_metrics", deleted)
	writeOK(w, map[string]any{"dry_run": false, "would_delete": count, "deleted": deleted})
}

// auditPrune registra uma operação de prune em change_log.
func (s *Server) auditPrune(r *http.Request, action string, deleted int64) {
	currentUser := auth.UserFromContext(r.Context())
	actor := "system"
	if currentUser != nil {
		actor = currentUser.Username
	}
	_, _ = s.db.AddChangeLog(r.Context(), db.ChangeLogEntry{
		Service:        "__prune__",
		Action:         action,
		Source:         "ui",
		User:           toNullString(actor),
		Status:         "completed",
		DockerResponse: toNullString("deleted=" + strconv.FormatInt(deleted, 10)),
	})
}
