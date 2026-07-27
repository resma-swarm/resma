// Package server — handlers internos para o ML sidecar.
//
// Estes endpoints (/api/internal/*) expõem dados do DuckDB para o ML sidecar
// Python via HTTP, evitando o lock exclusivo do DuckDB quando ambos processos
// sobem juntos. As rotas NÃO usam JWT — são acessadas apenas dentro da rede
// Docker pelo ML sidecar (http://go-dev:8080/api/internal/*).
package server

import (
	"net/http"
	"strconv"
	"time"
)

// registerInternalMLRoutes registra rotas /api/internal/* para o ML sidecar.
// Estas rotas NÃO usam JWT — são acessadas apenas dentro da rede Docker.
// Devem ser registradas em um mux separado sem auth middleware.
func (s *Server) registerInternalMLRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/internal/services/with-metrics", s.handleInternalServicesWithMetrics)
	mux.HandleFunc("GET /api/internal/services/{service}/metrics", s.handleInternalServiceMetrics)
	mux.HandleFunc("GET /api/internal/services/{service}/oom-count", s.handleInternalOOMCount)
	mux.HandleFunc("GET /api/internal/services/{service}/config", s.handleInternalServiceConfig)
	mux.HandleFunc("GET /api/internal/storage/volumes/metrics", s.handleInternalVolumeMetrics)
}

func (s *Server) handleInternalServicesWithMetrics(w http.ResponseWriter, r *http.Request) {
	// Suporte a ?minutes=N para filtrar apenas serviços ativos (janela curta).
	// Se minutes não for passado, usa days (default 7).
	if minutesStr := r.URL.Query().Get("minutes"); minutesStr != "" {
		minutes, err := strconv.Atoi(minutesStr)
		if err != nil || minutes <= 0 {
			writeError(w, http.StatusBadRequest, "invalid minutes")
			return
		}
		services, err := s.db.GetActiveServices(r.Context(), minutes)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if services == nil {
			services = []string{}
		}
		writeJSON(w, http.StatusOK, services)
		return
	}
	days := parseDaysQuery(r, 7)
	services, err := s.db.GetServicesWithMetrics(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if services == nil {
		services = []string{}
	}
	writeJSON(w, http.StatusOK, services)
}

func (s *Server) handleInternalServiceMetrics(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "missing service")
		return
	}
	days := parseDaysQuery(r, 7)
	rows, err := s.db.GetServiceMetricsRaw(r.Context(), service, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type metricPoint struct {
		TS         string  `json:"ts"`
		CPUPercent float64 `json:"cpu_percent"`
		MemUsage   int64   `json:"mem_usage"`
		MemLimit   int64   `json:"mem_limit"`
	}
	result := make([]metricPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, metricPoint{
			TS:         row.TS.Format(time.RFC3339Nano),
			CPUPercent: row.CPUPercent,
			MemUsage:   row.MemUsage,
			MemLimit:   row.MemLimit,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleInternalOOMCount(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "missing service")
		return
	}
	days := parseDaysQuery(r, 7)
	since := r.URL.Query().Get("since")
	var count int32
	var err error
	if since != "" {
		var t time.Time
		t, err = time.Parse(time.RFC3339, since)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since (use RFC3339)")
			return
		}
		count, err = s.db.GetOOMCountSince(r.Context(), service, t)
	} else {
		count, err = s.db.GetOOMCountByService(r.Context(), service, days)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int32{"count": count})
}

func (s *Server) handleInternalServiceConfig(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "missing service")
		return
	}
	cfg, err := s.db.GetServiceConfigRow(r.Context(), service)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cfg == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"cpu_limit":       0,
			"mem_limit":       0,
			"cpu_reservation": 0,
			"mem_reservation": 0,
			"template":        nil,
		})
		return
	}
	resp := map[string]any{
		"cpu_limit":       cfg.CPULimit.Float64,
		"mem_limit":       cfg.MemLimit.Int64,
		"cpu_reservation": cfg.CPUReservation.Float64,
		"mem_reservation": cfg.MemReservation.Int64,
	}
	if cfg.Template.Valid {
		resp["template"] = cfg.Template.String
	} else {
		resp["template"] = nil
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleInternalVolumeMetrics(w http.ResponseWriter, r *http.Request) {
	days := parseDaysQuery(r, 7)
	rows, err := s.db.GetVolumeMetricsRaw(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type volPoint struct {
		VolumeName string `json:"volume_name"`
		TS         string `json:"ts"`
		SizeBytes  int64  `json:"size_bytes"`
	}
	result := make([]volPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, volPoint{
			VolumeName: row.VolumeName,
			TS:         row.TS.Format(time.RFC3339Nano),
			SizeBytes:  row.SizeBytes,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// parseDaysQuery extrai ?days= da query string, com default.
func parseDaysQuery(r *http.Request, defaultDays int) int {
	d := r.URL.Query().Get("days")
	if d == "" {
		return defaultDays
	}
	n, err := strconv.Atoi(d)
	if err != nil || n <= 0 {
		return defaultDays
	}
	return n
}
