// Package server — /api/config, /api/oom-events, /api/change-log handlers.
package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/db"
)

// changeLogResponse é o contrato JSON exposto para o frontend (snake_case).
// O db.ChangeLogEntry usa sql.Null* que serializam como {"Float64":0.3,"Valid":true}
// e não tem tags JSON — este struct desembrulha para valores simples.
type changeLogResponse struct {
	ID                   int32    `json:"id"`
	Service              string   `json:"service"`
	Action               string   `json:"action"`
	Source               string   `json:"source"`
	ScheduleID           *int32   `json:"schedule_id"`
	CPULimitBefore       *float64 `json:"cpu_limit_before"`
	MemLimitBefore       *int64   `json:"mem_limit_before"`
	CPUReservationBefore *float64 `json:"cpu_reservation_before"`
	MemReservationBefore *int64   `json:"mem_reservation_before"`
	CPULimitAfter        *float64 `json:"cpu_limit_after"`
	MemLimitAfter        *int64   `json:"mem_limit_after"`
	CPUReservationAfter  *float64 `json:"cpu_reservation_after"`
	MemReservationAfter  *int64   `json:"mem_reservation_after"`
	User                 *string  `json:"user"`
	Status               string   `json:"status"`
	Error                *string  `json:"error"`
	DockerResponse       *string  `json:"docker_response"`
	CreatedAt            string   `json:"created_at"`
}

// toChangeLogResponse converte um db.ChangeLogEntry para o contrato da API.
func toChangeLogResponse(e db.ChangeLogEntry) changeLogResponse {
	resp := changeLogResponse{
		ID:        e.ID,
		Service:   e.Service,
		Action:    e.Action,
		Source:    e.Source,
		Status:    e.Status,
		CreatedAt: e.CreatedAt.Format(time.RFC3339Nano),
	}
	if e.ScheduleID.Valid {
		v := e.ScheduleID.Int32
		resp.ScheduleID = &v
	}
	if e.CPULimitBefore.Valid {
		v := e.CPULimitBefore.Float64
		resp.CPULimitBefore = &v
	}
	if e.MemLimitBefore.Valid {
		v := e.MemLimitBefore.Int64
		resp.MemLimitBefore = &v
	}
	if e.CPUReservationBefore.Valid {
		v := e.CPUReservationBefore.Float64
		resp.CPUReservationBefore = &v
	}
	if e.MemReservationBefore.Valid {
		v := e.MemReservationBefore.Int64
		resp.MemReservationBefore = &v
	}
	if e.CPULimitAfter.Valid {
		v := e.CPULimitAfter.Float64
		resp.CPULimitAfter = &v
	}
	if e.MemLimitAfter.Valid {
		v := e.MemLimitAfter.Int64
		resp.MemLimitAfter = &v
	}
	if e.CPUReservationAfter.Valid {
		v := e.CPUReservationAfter.Float64
		resp.CPUReservationAfter = &v
	}
	if e.MemReservationAfter.Valid {
		v := e.MemReservationAfter.Int64
		resp.MemReservationAfter = &v
	}
	if e.User.Valid && e.User.String != "" {
		v := e.User.String
		resp.User = &v
	}
	if e.Error.Valid && e.Error.String != "" {
		v := e.Error.String
		resp.Error = &v
	}
	if e.DockerResponse.Valid && e.DockerResponse.String != "" {
		v := e.DockerResponse.String
		resp.DockerResponse = &v
	}
	return resp
}

// toChangeLogResponses converte []db.ChangeLogEntry para []changeLogResponse.
func toChangeLogResponses(entries []db.ChangeLogEntry) []changeLogResponse {
	out := make([]changeLogResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toChangeLogResponse(e))
	}
	return out
}

// handleConfig retorna config operacional do RESMA.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"collect_interval":     int(s.cfg.CollectInterval.Seconds()),
		"retention_days":       s.cfg.RetentionDays,
		"analysis_window_days": s.cfg.AnalysisWindowDays,
	})
}

// handleOOMEvents lista eventos OOM, opcionalmente filtrando por service.
func (s *Server) handleOOMEvents(w http.ResponseWriter, r *http.Request) {
	service := queryValue(r, "service")
	rangeStr := queryValueDefault(r, "range", "7d")
	days := parseDays(rangeStr)
	data, err := s.buildOOMEvents(r.Context(), days, service)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, data)
}

// handleChangeLog lista change_log, opcionalmente por service.
func (s *Server) handleChangeLog(w http.ResponseWriter, r *http.Request) {
	service := queryValue(r, "service")
	limitStr := queryValueDefault(r, "limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 100
	}
	entries, err := s.buildChangeLog(r.Context(), service, int32(limit))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, entries)
}

// handleChangeLogByService lista change_log de um service específico.
func (s *Server) handleChangeLogByService(w http.ResponseWriter, r *http.Request) {
	service := pathValue(r, "service")
	limitStr := queryValueDefault(r, "limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 50
	}
	entries, err := s.db.GetChangeLog(r.Context(), service, int32(limit))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, toChangeLogResponses(entries))
}

// parseDays converte "7d" em 7, "30d" em 30. Default 7.
func parseDays(rangeStr string) int {
	if len(rangeStr) > 1 && rangeStr[len(rangeStr)-1] == 'd' {
		if d, err := strconv.Atoi(rangeStr[:len(rangeStr)-1]); err == nil {
			return d
		}
	}
	return 7
}

// scanOOMRows lê rows de oom_events e retorna slice de maps.
func scanOOMRows(rows *sql.Rows) []map[string]any {
	var out []map[string]any
	for rows.Next() {
		var ts time.Time
		var service, containerID, exitCode string
		if err := rows.Scan(&ts, &service, &containerID, &exitCode); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"ts":           ts.Format(time.RFC3339Nano),
			"service":      service,
			"container_id": containerID,
			"exit_code":    exitCode,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}
