// Package server — /api/config, /api/oom-events, /api/change-log handlers.
package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/db"
)

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
	if entries == nil {
		entries = []db.ChangeLogEntry{}
	}
	writeOK(w, entries)
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
