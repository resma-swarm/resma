// Package server — /api/schedules/* handlers.
//
// Portado de backend/routers/schedules.py.
package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/auth"
	"github.com/resma-swarm/resma/app/api/internal/db"
)

// scheduleResponse é o contrato JSON exposto para o frontend (snake_case).
// O db.Schedule usa sql.NullFloat64/sql.NullInt64/sql.NullTime/sql.NullString que
// serializam como {"Float64":0.3,"Valid":true} — este struct desembrulha para
// valores simples (number/null) que o React espera.
type scheduleResponse struct {
	ID             int32    `json:"id"`
	Service        string   `json:"service"`
	CPULimit       *float64 `json:"cpu_limit"`
	MemLimit       *int64   `json:"mem_limit"`
	CPUReservation *float64 `json:"cpu_reservation"`
	MemReservation *int64   `json:"mem_reservation"`
	ScheduledAt    string   `json:"scheduled_at"`
	Status         string   `json:"status"`
	AppliedAt      *string  `json:"applied_at"`
	Error          *string  `json:"error"`
	Attempts       int32    `json:"attempts"`
	CreatedAt      string   `json:"created_at"`
}

// toScheduleResponse converte um db.Schedule para o contrato da API.
// Campos sql.Null* são desembrulhados: Valid=true → ponteiro, Valid=false → nil.
func toScheduleResponse(s db.Schedule) scheduleResponse {
	resp := scheduleResponse{
		ID:          s.ID,
		Service:     s.Service,
		ScheduledAt: s.ScheduledAt.Format(time.RFC3339Nano),
		Status:      s.Status,
		Attempts:    s.Attempts,
		CreatedAt:   s.CreatedAt.Format(time.RFC3339Nano),
	}
	if s.CPULimit.Valid {
		v := s.CPULimit.Float64
		resp.CPULimit = &v
	}
	if s.MemLimit.Valid {
		v := s.MemLimit.Int64
		resp.MemLimit = &v
	}
	if s.CPUReservation.Valid {
		v := s.CPUReservation.Float64
		resp.CPUReservation = &v
	}
	if s.MemReservation.Valid {
		v := s.MemReservation.Int64
		resp.MemReservation = &v
	}
	if s.AppliedAt.Valid {
		v := s.AppliedAt.Time.Format(time.RFC3339Nano)
		resp.AppliedAt = &v
	}
	if s.Error.Valid && s.Error.String != "" {
		v := s.Error.String
		resp.Error = &v
	}
	return resp
}

// toScheduleResponses converte []db.Schedule para []scheduleResponse.
func toScheduleResponses(schedules []db.Schedule) []scheduleResponse {
	out := make([]scheduleResponse, 0, len(schedules))
	for _, s := range schedules {
		out = append(out, toScheduleResponse(s))
	}
	return out
}

// registerScheduleRoutes registra as rotas de schedules.
// Fase 8: rotas de escrita (POST/DELETE) exigem role owner ou admin.
func (s *Server) registerScheduleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/schedules", s.handleListSchedules)
	mux.HandleFunc("GET /api/schedules/pending", s.handleListPendingSchedules)
	mux.HandleFunc("GET /api/schedules/history", s.handleScheduleHistory)

	// Rotas de escrita — owner/admin apenas
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("POST /api/schedules", rbac(http.HandlerFunc(s.handleCreateSchedule)))
	mux.Handle("DELETE /api/schedules/{schedule_id}", rbac(http.HandlerFunc(s.handleCancelSchedule)))
}

// handleListSchedules lista agendamentos, opcionalmente por status.
func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	status := queryValue(r, "status")
	schedules, err := s.buildSchedulesList(r.Context(), status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, schedules)
}

// handleListPendingSchedules lista agendamentos pendentes (incluindo futuros).
// Usado pelo frontend para mostrar ícone de schedule nos cards.
func (s *Server) handleListPendingSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.db.ListSchedules(r.Context(), "pending")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, toScheduleResponses(schedules))
}

// handleScheduleHistory retorna histórico de agendamentos.
func (s *Server) handleScheduleHistory(w http.ResponseWriter, r *http.Request) {
	service := queryValue(r, "service")
	limitStr := queryValueDefault(r, "limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 50
	}
	history, err := s.db.GetScheduleHistory(r.Context(), service, int32(limit))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, toScheduleResponses(history))
}

// handleCreateSchedule cria um novo agendamento.
func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Service        string  `json:"service"`
		CPULimit       float64 `json:"cpu_limit"`
		MemLimit       int64   `json:"mem_limit"`
		CPUReservation float64 `json:"cpu_reservation"`
		MemReservation int64   `json:"mem_reservation"`
		ScheduledAt    string  `json:"scheduled_at"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Service == "" {
		writeError(w, http.StatusBadRequest, "service required")
		return
	}
	if err := s.validateService(r, req.Service); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	existing, _ := s.db.GetPendingScheduleForService(r.Context(), req.Service)
	if existing != nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("Já existe um agendamento pendente para '%s'", req.Service))
		return
	}
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Formato de data inválido. Use ISO 8601.")
		return
	}
	if scheduledAt.Before(time.Now()) {
		writeError(w, http.StatusUnprocessableEntity, "A data agendada deve ser no futuro.")
		return
	}
	schID, err := s.db.CreateSchedule(r.Context(), db.Schedule{
		Service:        req.Service,
		CPULimit:       sql.NullFloat64{Float64: req.CPULimit, Valid: true},
		MemLimit:       sql.NullInt64{Int64: req.MemLimit, Valid: true},
		CPUReservation: sql.NullFloat64{Float64: req.CPUReservation, Valid: true},
		MemReservation: sql.NullInt64{Int64: req.MemReservation, Valid: true},
		ScheduledAt:    scheduledAt,
	})
	if err != nil || schID == 0 {
		writeError(w, http.StatusInternalServerError, "Falha ao criar agendamento")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":      schID,
		"message": fmt.Sprintf("Agendamento criado para '%s' em %s", req.Service, scheduledAt.Format(time.RFC3339)),
	})
}

// handleCancelSchedule cancela um agendamento pendente.
func (s *Server) handleCancelSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := pathValue(r, "schedule_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid schedule id")
		return
	}
	schedule, err := s.db.GetScheduleByID(r.Context(), int32(id))
	if err != nil || schedule == nil {
		writeError(w, http.StatusNotFound, "Agendamento não encontrado")
		return
	}
	if schedule.Status != "pending" {
		writeError(w, http.StatusConflict, fmt.Sprintf("Não é possível cancelar um agendamento com status '%s'", schedule.Status))
		return
	}
	cancelled, err := s.db.CancelSchedule(r.Context(), int32(id))
	if err != nil || !cancelled {
		writeError(w, http.StatusConflict, "Agendamento não pôde ser cancelado")
		return
	}
	writeOK(w, map[string]any{"success": true, "message": "Agendamento cancelado"})
}
