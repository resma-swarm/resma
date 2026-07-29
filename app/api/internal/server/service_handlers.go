// Package server — /api/services/* handlers.
//
// Portado de backend/routers/services.py.
package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/auth"
	"github.com/resma-swarm/resma/app/api/internal/docker"
)

// registerServiceRoutes registra as rotas de services no mux interno.
// Fase 8: rotas de escrita (PATCH archive/restore) exigem role owner ou admin.
func (s *Server) registerServiceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/services", s.handleListServices)
	mux.HandleFunc("GET /api/services/sparklines", s.handleServiceSparklines)
	mux.HandleFunc("GET /api/services/{name}/metrics", s.handleServiceMetrics)
	mux.HandleFunc("GET /api/services/{name}/stats", s.handleServiceStats)
	mux.HandleFunc("GET /api/services/{name}/containers", s.handleServiceContainers)
	mux.HandleFunc("GET /api/services/containers/{container_id}/metrics", s.handleContainerMetrics)
	mux.HandleFunc("GET /api/services/containers/{container_id}/stats", s.handleContainerStats)
	mux.HandleFunc("GET /api/services/containers/{container_id}/network-info", s.handleContainerNetworkInfo)

	// Rotas de escrita — owner/admin apenas
	rbac := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
	mux.Handle("PATCH /api/services/{name}/archive", rbac(http.HandlerFunc(s.handleArchiveService)))
	mux.Handle("PATCH /api/services/{name}/restore", rbac(http.HandlerFunc(s.handleRestoreService)))
}

// handleListServices lista todos os serviços com métricas agregadas.
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	result, err := s.buildServicesList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// handleArchiveService arquiva um serviço.
func (s *Server) handleArchiveService(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	if err := s.db.SetServiceArchived(r.Context(), name, true); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]any{"success": true, "message": fmt.Sprintf("Serviço '%s' arquivado", name)})
}

// handleRestoreService restaura um serviço arquivado.
func (s *Server) handleRestoreService(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	if err := s.db.SetServiceArchived(r.Context(), name, false); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, map[string]any{"success": true, "message": fmt.Sprintf("Serviço '%s' restaurado", name)})
}

// handleServiceSparklines retorna sparklines de métricas por serviço.
func (s *Server) handleServiceSparklines(w http.ResponseWriter, r *http.Request) {
	pointsStr := queryValueDefault(r, "points", "20")
	points, _ := atoiSafe(pointsStr)
	if points < 1 {
		points = 1
	}
	if points > 100 {
		points = 100
	}

	result, err := s.buildServiceSparklines(r.Context(), points)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// handleServiceMetrics retorna métricas temporais de um serviço.
//
// Downsampling: se o serviço tem mais de 500 amostras no período, agrega
// em buckets temporais para limitar a ~300 pontos. Um gráfico de 7 dias
// com 68000 pontos é ilegível e causa picos de CPU/memória no browser
// (recharts renderiza cada ponto). 300 pontos é suficiente para visualização.
func (s *Server) handleServiceMetrics(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	rangeStr := queryValueDefault(r, "range", "7d")
	days := parseDays(rangeStr)

	// Contar amostras primeiro
	var totalSamples int
	countQuery := fmt.Sprintf(`
		SELECT count(*) FROM metrics
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days)
	_ = s.db.QueryRowContext(r.Context(), countQuery, name).Scan(&totalSamples)

	const maxPoints = 300

	if totalSamples <= maxPoints {
		// Poucas amostras — retornar tudo
		query := fmt.Sprintf(`
			SELECT ts, cpu_percent, mem_usage, mem_limit, mem_percent
			FROM metrics
			WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
			ORDER BY ts`, days)
		rows, err := s.db.QueryContext(r.Context(), query, name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		result := []map[string]any{}
		for rows.Next() {
			var ts time.Time
			var cpu, memUsage, memLimit, memPercent float64
			if err := rows.Scan(&ts, &cpu, &memUsage, &memLimit, &memPercent); err != nil {
				continue
			}
			result = append(result, map[string]any{
				"ts":          ts.Format(time.RFC3339Nano),
				"cpu_percent": cpu,
				"mem_usage":   int64(memUsage),
				"mem_limit":   int64(memLimit),
				"mem_percent": memPercent,
			})
		}
		writeOK(w, result)
		return
	}

	// Muitas amostras — agregar em buckets temporais
	// bucketSeconds = (days * 86400) / maxPoints
	bucketSeconds := (days * 86400) / maxPoints
	if bucketSeconds < 60 {
		bucketSeconds = 60 // mínimo 1 minuto
	}

	query := fmt.Sprintf(`
		SELECT
			time_bucket(INTERVAL '%d seconds', ts) as bucket_ts,
			avg(cpu_percent) as cpu_percent,
			avg(mem_usage) as mem_usage,
			max(mem_limit) as mem_limit,
			avg(mem_percent) as mem_percent
		FROM metrics
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY bucket_ts
		ORDER BY bucket_ts`, bucketSeconds, days)
	rows, err := s.db.QueryContext(r.Context(), query, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var ts time.Time
		var cpu, memUsage, memLimit, memPercent float64
		if err := rows.Scan(&ts, &cpu, &memUsage, &memLimit, &memPercent); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"ts":          ts.Format(time.RFC3339Nano),
			"cpu_percent": cpu,
			"mem_usage":   int64(memUsage),
			"mem_limit":   int64(memLimit),
			"mem_percent": memPercent,
		})
	}
	writeOK(w, result)
}

// handleServiceStats retorna estatísticas agregadas de um serviço.
func (s *Server) handleServiceStats(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	days := s.cfg.AnalysisWindowDays

	query := fmt.Sprintf(`
		SELECT
			count(*) as samples,
			quantile(cpu_percent, 0.50) as cpu_p50,
			quantile(cpu_percent, 0.95) as cpu_p95,
			min(cpu_percent) as cpu_min,
			max(cpu_percent) as cpu_max,
			avg(cpu_percent) as cpu_avg,
			quantile(mem_usage, 0.50) as mem_p50,
			quantile(mem_usage, 0.99) as mem_p99,
			min(mem_usage) as mem_min,
			max(mem_usage) as mem_max,
			avg(mem_usage) as mem_avg
		FROM metrics
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days)
	var samples int32
	var cpuP50, cpuP95, cpuMin, cpuMax, cpuAvg, memP50, memP99, memMin, memMax, memAvg float64
	err := s.db.QueryRowContext(r.Context(), query, name).
		Scan(&samples, &cpuP50, &cpuP95, &cpuMin, &cpuMax, &cpuAvg,
			&memP50, &memP99, &memMin, &memMax, &memAvg)
	if err != nil || samples == 0 {
		writeError(w, http.StatusNotFound, "No data for service")
		return
	}
	writeOK(w, map[string]any{
		"service": name,
		"samples": samples,
		"cpu_p50": round2(cpuP50),
		"cpu_p95": round2(cpuP95),
		"cpu_min": round2(cpuMin),
		"cpu_max": round2(cpuMax),
		"cpu_avg": round2(cpuAvg),
		"mem_p50": int64(memP50),
		"mem_p99": int64(memP99),
		"mem_min": int64(memMin),
		"mem_max": int64(memMax),
		"mem_avg": round2(memAvg),
	})
}

// handleServiceContainers retorna containers de um serviço com stats.
func (s *Server) handleServiceContainers(w http.ResponseWriter, r *http.Request) {
	name := pathValue(r, "name")
	days := s.cfg.AnalysisWindowDays
	ctx := r.Context()

	query := fmt.Sprintf(`
		SELECT container_id,
			   count(*) as samples,
			   quantile(cpu_percent, 0.50) as cpu_p50,
			   quantile(cpu_percent, 0.95) as cpu_p95,
			   min(cpu_percent) as cpu_min,
			   max(cpu_percent) as cpu_max,
			   avg(cpu_percent) as cpu_avg,
			   quantile(mem_usage, 0.50) as mem_p50,
			   quantile(mem_usage, 0.99) as mem_p99,
			   min(mem_usage) as mem_min,
			   max(mem_usage) as mem_max,
			   avg(mem_usage) as mem_avg,
			   MAX(ts) as last_seen
		FROM metrics
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY container_id ORDER BY last_seen DESC`, days)
	rows, err := s.db.QueryContext(ctx, query, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	runningIDs, _ := s.docker.GetRunningContainerIDs(ctx)
	allContainers := s.docker.ListContainers()
	containerNetworksMap := make(map[string]map[string]string)
	for _, c := range allContainers {
		containerNetworksMap[c.ID] = c.Networks
	}

	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	result := []map[string]any{}
	for rows.Next() {
		var cid string
		var samples int32
		var cpuP50, cpuP95, cpuMin, cpuMax, cpuAvg, memP50, memP99, memMin, memMax, memAvg float64
		var lastSeen sql.NullTime
		if err := rows.Scan(&cid, &samples, &cpuP50, &cpuP95, &cpuMin, &cpuMax, &cpuAvg,
			&memP50, &memP99, &memMin, &memMax, &memAvg, &lastSeen); err != nil {
			continue
		}
		status := "legado"
		if runningIDs[cid] {
			status = "online"
		} else if lastSeen.Valid && lastSeen.Time.After(fiveMinAgo) {
			status = "offline"
		}
		lastSeenStr := ""
		if lastSeen.Valid {
			lastSeenStr = lastSeen.Time.Format(time.RFC3339Nano)
		}
		networks := containerNetworksMap[cid]
		if networks == nil {
			networks = map[string]string{}
		}
		result = append(result, map[string]any{
			"container_id": cid,
			"samples":      samples,
			"cpu_p50":      round2(cpuP50),
			"cpu_p95":      round2(cpuP95),
			"cpu_min":      round2(cpuMin),
			"cpu_max":      round2(cpuMax),
			"cpu_avg":      round2(cpuAvg),
			"mem_p50":      int64(memP50),
			"mem_p99":      int64(memP99),
			"mem_min":      int64(memMin),
			"mem_max":      int64(memMax),
			"mem_avg":      round2(memAvg),
			"last_seen":    nullStr(lastSeenStr, !lastSeen.Valid),
			"status":       status,
			"networks":     networks,
		})
	}
	writeOK(w, result)
}

// handleContainerMetrics retorna métricas temporais de um container.
func (s *Server) handleContainerMetrics(w http.ResponseWriter, r *http.Request) {
	cid := pathValue(r, "container_id")
	rangeStr := queryValueDefault(r, "range", "7d")
	days := parseDays(rangeStr)

	query := fmt.Sprintf(`
		SELECT ts, cpu_percent, mem_usage, mem_limit, mem_percent
		FROM metrics
		WHERE container_id = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
		ORDER BY ts`, days)
	rows, err := s.db.QueryContext(r.Context(), query, cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var ts time.Time
		var cpu, memUsage, memLimit, memPercent float64
		if err := rows.Scan(&ts, &cpu, &memUsage, &memLimit, &memPercent); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"ts":          ts.Format(time.RFC3339Nano),
			"cpu_percent": cpu,
			"mem_usage":   int64(memUsage),
			"mem_limit":   int64(memLimit),
			"mem_percent": memPercent,
		})
	}
	writeOK(w, result)
}

// handleContainerStats retorna estatísticas agregadas de um container.
func (s *Server) handleContainerStats(w http.ResponseWriter, r *http.Request) {
	cid := pathValue(r, "container_id")
	days := s.cfg.AnalysisWindowDays

	query := fmt.Sprintf(`
		SELECT
			count(*) as samples,
			quantile(cpu_percent, 0.50) as cpu_p50,
			quantile(cpu_percent, 0.95) as cpu_p95,
			min(cpu_percent) as cpu_min,
			max(cpu_percent) as cpu_max,
			avg(cpu_percent) as cpu_avg,
			quantile(mem_usage, 0.50) as mem_p50,
			quantile(mem_usage, 0.99) as mem_p99,
			min(mem_usage) as mem_min,
			max(mem_usage) as mem_max,
			avg(mem_usage) as mem_avg
		FROM metrics
		WHERE container_id = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days)
	var samples int32
	var cpuP50, cpuP95, cpuMin, cpuMax, cpuAvg, memP50, memP99, memMin, memMax, memAvg float64
	err := s.db.QueryRowContext(r.Context(), query, cid).
		Scan(&samples, &cpuP50, &cpuP95, &cpuMin, &cpuMax, &cpuAvg,
			&memP50, &memP99, &memMin, &memMax, &memAvg)
	if err != nil || samples == 0 {
		writeError(w, http.StatusNotFound, "No data for container")
		return
	}
	writeOK(w, map[string]any{
		"container_id": cid,
		"samples":      samples,
		"cpu_p50":      round2(cpuP50),
		"cpu_p95":      round2(cpuP95),
		"cpu_min":      round2(cpuMin),
		"cpu_max":      round2(cpuMax),
		"cpu_avg":      round2(cpuAvg),
		"mem_p50":      int64(memP50),
		"mem_p99":      int64(memP99),
		"mem_min":      int64(memMin),
		"mem_max":      int64(memMax),
		"mem_avg":      round2(memAvg),
	})
}

// handleContainerNetworkInfo retorna networks de um container.
func (s *Server) handleContainerNetworkInfo(w http.ResponseWriter, r *http.Request) {
	cid := pathValue(r, "container_id")
	networks, err := s.docker.GetContainerNetworks(r.Context(), cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if networks == nil {
		networks = []docker.ContainerNetwork{}
	}
	writeOK(w, networks)
}

// --- helpers ---

// nullStr retorna nil se isNull, ou o string.
func nullStr(s string, isNull bool) any {
	if isNull {
		return nil
	}
	return s
}

// atoiSafe converte string para int com fallback 0.
func atoiSafe(s string) (int, error) {
	return strconv.Atoi(s)
}

// round2 arredonda para 2 casas decimais.
func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
