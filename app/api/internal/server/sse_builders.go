// Package server — funções build* para SSE payload completo.
//
// Estas funções extraem a lógica de build de dados dos handlers HTTP,
// permitindo que o collector e agent_handlers publiquem o MESMO payload
// do GET via SSE — eliminando a necessidade de refetch no frontend.
//
// Cada build* retorna (data, error) sem escrever no ResponseWriter.
// Os handlers HTTP chamam build* + writeOK; os publishers SSE chamam
// build* + sse.Publish. Zero duplicação de lógica.
package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/docker"
)

// buildDashboardData constrói o payload completo de /api/dashboard.
// Usado por: handleDashboard (GET) + collector (SSE topic metrics/dashboard).
func (s *Server) buildDashboardData(ctx context.Context) (map[string]any, error) {
	days := s.cfg.AnalysisWindowDays

	query := fmt.Sprintf(`
		SELECT service, count(DISTINCT container_id) as containers,
		       quantile(cpu_percent, 0.95) as cpu_p95,
		       quantile(mem_usage, 0.99) as mem_p99
		FROM metrics
		WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY service`, days)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type svcMetric struct {
		Name   string
		Count  int32
		CPUP95 float64
		MemP99 float64
	}
	services := []svcMetric{}
	for rows.Next() {
		var sm svcMetric
		if err := rows.Scan(&sm.Name, &sm.Count, &sm.CPUP95, &sm.MemP99); err != nil {
			continue
		}
		services = append(services, sm)
	}

	// activeContainersQuery conta containers ATIVOS (últimos 5 min) por serviço.
	// Usado para o card "Containers" do dashboard — antes somava count(DISTINCT
	// container_id) da janela de 7d (acumulado histórico), mostrando 239 para
	// ~12 containers reais. Ver BUG-003 do QA Playwright 2026-08-06.
	activeContainersQuery := `SELECT service, count(DISTINCT container_id) FROM metrics WHERE ts > now()::TIMESTAMP - INTERVAL 5 MINUTE GROUP BY service`
	activeContainersRows, err := s.db.QueryContext(ctx, activeContainersQuery)
	if err != nil {
		return nil, err
	}
	activeContainerCount := make(map[string]int32)
	for activeContainersRows.Next() {
		var svc string
		var cnt int32
		if err := activeContainersRows.Scan(&svc, &cnt); err != nil {
			continue
		}
		activeContainerCount[svc] = cnt
	}
	activeContainersRows.Close()

	activeQuery := `SELECT DISTINCT service FROM metrics WHERE ts > now()::TIMESTAMP - INTERVAL 5 MINUTE`
	activeRows, err := s.db.QueryContext(ctx, activeQuery)
	if err != nil {
		return nil, err
	}
	activeSet := make(map[string]bool)
	for activeRows.Next() {
		var svc string
		if err := activeRows.Scan(&svc); err != nil {
			continue
		}
		activeSet[svc] = true
	}
	activeRows.Close()

	activeServices := make([]svcMetric, 0, len(services))
	for _, sm := range services {
		if activeSet[sm.Name] {
			activeServices = append(activeServices, sm)
		}
	}
	services = activeServices

	// totalContainers usa a contagem de containers ativos (últimos 5 min),
	// não o acumulado histórico da janela de análise.
	totalContainers := int32(0)
	for s := range activeContainerCount {
		totalContainers += activeContainerCount[s]
	}

	topCPU := make([]svcMetric, len(services))
	copy(topCPU, services)
	sort.Slice(topCPU, func(i, j int) bool { return topCPU[i].CPUP95 > topCPU[j].CPUP95 })
	if len(topCPU) > 5 {
		topCPU = topCPU[:5]
	}

	topMem := make([]svcMetric, len(services))
	copy(topMem, services)
	sort.Slice(topMem, func(i, j int) bool { return topMem[i].MemP99 > topMem[j].MemP99 })
	if len(topMem) > 5 {
		topMem = topMem[:5]
	}

	// OOM count deve bater com /api/alerts (alert_handlers.go): mesma query
	// GROUP BY ts, service com LIMIT 50. Antes contava todos os grupos (354),
	// divergindo da página de Alerts (50) — ver BUG-001 do QA Playwright.
	oomCountQuery := fmt.Sprintf(`
		SELECT count(*) FROM (
			SELECT ts, service FROM oom_events
			WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS
			GROUP BY ts, service
			ORDER BY ts DESC
			LIMIT 50
		)`, days)
	var oomCount int32
	oomCountRow := s.db.QueryRowContext(ctx, oomCountQuery)
	_ = oomCountRow.Scan(&oomCount)

	leakCount, driftCount := 0, 0
	if alerts, err := s.ml.GetAlerts(ctx); err == nil && alerts != nil {
		leakCount = len(alerts.LeakAlerts)
		driftCount = len(alerts.DriftAlerts)
	}
	alertsSummary := map[string]any{
		"total":       int(oomCount) + leakCount + driftCount,
		"oom_count":   int(oomCount),
		"leak_count":  leakCount,
		"drift_count": driftCount,
	}

	cluster, _ := s.db.GetCluster(ctx)
	clusterMap := map[string]any{}
	if cluster != nil {
		clusterMap = map[string]any{
			"id":             cluster.ID,
			"nodes_total":    cluster.NodesTotal,
			"managers_total": cluster.ManagersTotal,
			"workers_total":  cluster.WorkersTotal,
			"nodes_ready":    cluster.NodesReady,
			"nodes_down":     cluster.NodesDown,
			"quorum_healthy": cluster.QuorumHealthy,
			"self_node_id":   cluster.SelfNodeID,
			"warnings":       parseJSONWarnings(cluster.Warnings),
			"updated_at":     formatTime(cluster.UpdatedAt),
		}
	}

	clusterCapacity, _ := s.db.GetClusterSummary(ctx, days)

	nodes, _ := s.db.GetNodes(ctx)
	nodesDist := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		nodesDist = append(nodesDist, map[string]any{
			"hostname":      n.Hostname,
			"node_id":       n.NodeID,
			"role":          n.Role,
			"status":        n.Status,
			"tasks_running": n.TasksRunning,
		})
	}

	topCPUOut := make([]map[string]any, 0, len(topCPU))
	for _, sm := range topCPU {
		topCPUOut = append(topCPUOut, map[string]any{
			"name":            sm.Name,
			"container_count": sm.Count,
			"cpu_p95":         round2(sm.CPUP95),
			"mem_p99":         int64(sm.MemP99),
		})
	}
	topMemOut := make([]map[string]any, 0, len(topMem))
	for _, sm := range topMem {
		topMemOut = append(topMemOut, map[string]any{
			"name":            sm.Name,
			"container_count": sm.Count,
			"cpu_p95":         round2(sm.CPUP95),
			"mem_p99":         int64(sm.MemP99),
		})
	}

	return map[string]any{
		"total_services":     len(services),
		"total_containers":   totalContainers,
		"top_cpu_consumers":  topCPUOut,
		"top_mem_consumers":  topMemOut,
		"alerts_summary":     alertsSummary,
		"cluster":            clusterMap,
		"cluster_capacity":   clusterCapacity,
		"nodes_distribution": nodesDist,
	}, nil
}

// buildServicesList constrói o payload completo de /api/services.
// Usado por: handleListServices (GET) + collector (SSE topic services/metrics).
//
// Cada serviço inclui um risk_score (0-100) calculado por fórmula determinística
// baseada em benchmarks da indústria (Kubernetes VPA, Linux OOM killer, Datadog):
//   - Memory pressure (40%): mem_p99 / mem_limit
//   - OOM history (30%): oom_events count nos últimos N dias
//   - Memory leak (20%): ML sidecar detecta leak
//   - Config drift (10%): ML sidecar detecta drift
func (s *Server) buildServicesList(ctx context.Context) ([]map[string]any, error) {
	registry, err := s.db.GetServiceRegistry(ctx)
	if err != nil {
		return nil, err
	}

	days := s.cfg.AnalysisWindowDays
	query := fmt.Sprintf(`
		SELECT service, count(DISTINCT container_id) as containers,
		       quantile(cpu_percent, 0.95) as cpu_p95,
		       quantile(mem_usage, 0.99) as mem_p99,
		       MAX(ts) as last_seen
		FROM metrics
		WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY service ORDER BY cpu_p95 DESC`, days)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activeQuery := `SELECT service, count(DISTINCT container_id) as active
		FROM metrics WHERE ts > now()::TIMESTAMP - INTERVAL 5 MINUTES GROUP BY service`
	activeRows, err := s.db.QueryContext(ctx, activeQuery)
	if err != nil {
		return nil, err
	}
	activeMap := make(map[string]int32)
	for activeRows.Next() {
		var svc string
		var cnt int32
		if err := activeRows.Scan(&svc, &cnt); err != nil {
			continue
		}
		activeMap[svc] = cnt
	}
	activeRows.Close()

	// OOM count por serviço (1 query aggregada, não 1 por serviço)
	oomMap := make(map[string]int32)
	oomQuery := fmt.Sprintf(`
		SELECT service, count(*) as oom_count
		FROM oom_events
		WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY service`, days)
	oomRows, err := s.db.QueryContext(ctx, oomQuery)
	if err == nil {
		for oomRows.Next() {
			var svc string
			var cnt int32
			if err := oomRows.Scan(&svc, &cnt); err != nil {
				continue
			}
			oomMap[svc] = cnt
		}
		oomRows.Close()
	}

	// Leak/Drift do ML sidecar (cached, fallback graceful)
	leakMap := make(map[string]bool)
	driftMap := make(map[string]bool)
	if alerts, err := s.ml.GetAlerts(ctx); err == nil && alerts != nil {
		for _, a := range alerts.LeakAlerts {
			if svc, ok := a["service"].(string); ok {
				leakMap[svc] = true
			}
		}
		for _, a := range alerts.DriftAlerts {
			if svc, ok := a["service"].(string); ok {
				driftMap[svc] = true
			}
		}
	}

	resourcesMap, _ := s.docker.GetAllServiceResources(ctx)

	var result []map[string]any
	for rows.Next() {
		var svcName string
		var containers int32
		var cpuP95, memP99 float64
		var lastSeen sql.NullTime
		if err := rows.Scan(&svcName, &containers, &cpuP95, &memP99, &lastSeen); err != nil {
			continue
		}
		reg, exists := registry[svcName]
		regStatus := ""
		if exists {
			regStatus = reg.Status
		}
		if regStatus == "archived" {
			continue
		}
		status := regStatus
		if status != "online" && status != "offline" && status != "legado" {
			status = "legado"
		}
		lastSeenStr := ""
		if lastSeen.Valid {
			lastSeenStr = lastSeen.Time.Format(time.RFC3339Nano)
		}
		current := map[string]any{
			"cpu_limit":       float64(0),
			"mem_limit":       int64(0),
			"cpu_reservation": float64(0),
			"mem_reservation": int64(0),
		}
		if res, ok := resourcesMap[svcName]; ok {
			current = map[string]any{
				"cpu_limit":       res.CPULimit,
				"mem_limit":       res.MemLimit,
				"cpu_reservation": res.CPUReservation,
				"mem_reservation": res.MemReservation,
			}
		}

		// Risk score determinístico (0-100)
		oomCount := int(oomMap[svcName])
		hasLeak := leakMap[svcName]
		hasDrift := driftMap[svcName]
		memLimit := current["mem_limit"].(int64)
		riskScore := calcRiskScore(memP99, memLimit, oomCount, hasLeak, hasDrift)

		result = append(result, map[string]any{
			"name":              svcName,
			"container_count":   containers,
			"active_containers": activeMap[svcName],
			"cpu_p95":           round2(cpuP95),
			"mem_p99":           int64(memP99),
			"last_seen":         nullStr(lastSeenStr, !lastSeen.Valid),
			"status":            status,
			"current":           current,
			"risk_score":        riskScore,
			"risk_factors": map[string]any{
				"oom_count": oomCount,
				"has_leak":  hasLeak,
				"has_drift": hasDrift,
				"mem_limit": memLimit,
			},
		})
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

// buildServiceSparklines constrói o payload de /api/services/sparklines.
// Usado por: handleServiceSparklines (GET) + collector (SSE topic services/metrics).
func (s *Server) buildServiceSparklines(ctx context.Context, points int) (map[string][]map[string]any, error) {
	if points < 1 {
		points = 1
	}
	if points > 100 {
		points = 100
	}

	registry, _ := s.db.GetServiceRegistry(ctx)
	archived := make(map[string]bool)
	for name, reg := range registry {
		if reg.Status == "archived" {
			archived[name] = true
		}
	}

	query := fmt.Sprintf(`
		SELECT service, ts, cpu_percent, mem_usage
		FROM (
			SELECT service, ts, cpu_percent, mem_usage,
			       ROW_NUMBER() OVER (PARTITION BY service ORDER BY ts DESC) as rn
			FROM metrics
			WHERE ts > now()::TIMESTAMP - INTERVAL 1 DAYS
		) t
		WHERE rn <= %d
		ORDER BY service, ts`, points)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]map[string]any)
	for rows.Next() {
		var svc string
		var ts time.Time
		var cpu, mem float64
		if err := rows.Scan(&svc, &ts, &cpu, &mem); err != nil {
			continue
		}
		if archived[svc] {
			continue
		}
		result[svc] = append(result[svc], map[string]any{
			"ts":  ts.Format(time.RFC3339Nano),
			"cpu": round2(cpu),
			"mem": int64(mem),
		})
	}
	return result, nil
}

// buildNodesList constrói o payload de /api/nodes.
// Usado por: handleListNodes (GET) + collector (SSE topic nodes/agents).
func (s *Server) buildNodesList(ctx context.Context) ([]map[string]any, error) {
	nodes, err := s.db.GetNodes(ctx)
	if err != nil {
		return nil, err
	}
	days := s.cfg.AnalysisWindowDays
	result := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		consumption, _ := s.db.GetNodeConsumption(ctx, n.NodeID, days)
		labels := map[string]any{}
		if n.Labels != "" {
			_ = json.Unmarshal([]byte(n.Labels), &labels)
		}
		result = append(result, map[string]any{
			"node_id":        n.NodeID,
			"hostname":       n.Hostname,
			"role":           n.Role,
			"availability":   n.Availability,
			"status":         n.Status,
			"address":        n.Address,
			"cpu_total":      n.CPUTotal,
			"mem_total":      n.MemTotal,
			"os":             n.OS,
			"architecture":   n.Architecture,
			"engine_version": n.EngineVersion,
			"is_leader":      n.IsLeader,
			"reachability":   n.Reachability,
			"labels":         labels,
			"tasks_running":  n.TasksRunning,
			"cpu_p95":        consumption.CPUP95,
			"mem_p99":        consumption.MemP99,
			"containers":     consumption.Containers,
			"updated_at":     formatTime(n.UpdatedAt),
		})
	}
	return result, nil
}

// buildClusterInfo constrói o payload de /api/nodes/cluster.
// Usado por: handleClusterInfo (GET) + collector (SSE topic dashboard).
func (s *Server) buildClusterInfo(ctx context.Context) (map[string]any, error) {
	cluster, err := s.db.GetCluster(ctx)
	if err != nil || cluster == nil {
		return map[string]any{
			"id":             "",
			"nodes_total":    0,
			"managers_total": 0,
			"workers_total":  0,
			"nodes_ready":    0,
			"nodes_down":     0,
			"quorum_healthy": false,
			"self_node_id":   "",
			"warnings":       []any{},
			"updated_at":     nil,
		}, nil
	}
	warnings := []any{}
	if cluster.Warnings != "" {
		_ = json.Unmarshal([]byte(cluster.Warnings), &warnings)
	}
	return map[string]any{
		"id":             cluster.ID,
		"nodes_total":    cluster.NodesTotal,
		"managers_total": cluster.ManagersTotal,
		"workers_total":  cluster.WorkersTotal,
		"nodes_ready":    cluster.NodesReady,
		"nodes_down":     cluster.NodesDown,
		"quorum_healthy": cluster.QuorumHealthy,
		"self_node_id":   cluster.SelfNodeID,
		"warnings":       warnings,
		"updated_at":     formatTime(cluster.UpdatedAt),
	}, nil
}

// buildStorageSummary constrói o payload de /api/storage/summary.
// Usado por: handleStorageSummary (GET) + collector (SSE topic dashboard).
func (s *Server) buildStorageSummary(ctx context.Context) (map[string]any, error) {
	df, err := s.docker.GetSystemDF(ctx)
	if err != nil {
		return nil, err
	}
	latest, _ := s.db.GetLatestStorageSummary(ctx)
	return map[string]any{
		"live":            df,
		"latest_snapshot": latest,
	}, nil
}

// buildTasksList constrói o payload de /api/tasks.
// Usado por: handleListTasks (GET) + collector (SSE topic tasks).
func (s *Server) buildTasksList(ctx context.Context) ([]map[string]any, error) {
	tasks, err := s.db.GetTasks(ctx)
	if err != nil {
		return nil, err
	}
	nodes, _ := s.db.GetNodes(ctx)
	nodeHostnames := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeHostnames[n.NodeID] = n.Hostname
	}

	result := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		hostname := nodeHostnames[t.NodeID]
		result = append(result, map[string]any{
			"task_id":       t.TaskID,
			"service":       t.Service,
			"node_id":       t.NodeID,
			"node_hostname": hostname,
			"slot":          t.Slot,
			"status":        t.Status,
			"desired_state": t.DesiredState,
			"created_at":    formatTimePtr(t.CreatedAt),
			"updated_at":    formatTimePtr(t.UpdatedAt),
		})
	}
	return result, nil
}

// buildServicesHealth constrói o payload de /api/services/health.
// Usado por: handleServicesHealth (GET) + collector (SSE topic tasks).
func (s *Server) buildServicesHealth(ctx context.Context, days int) ([]ServiceHealth, error) {
	tasks, err := s.db.GetTasks(ctx)
	if err != nil {
		return nil, err
	}

	healthMap := make(map[string]*ServiceHealth)
	for _, t := range tasks {
		if t.Service == "" {
			continue
		}
		h, ok := healthMap[t.Service]
		if !ok {
			h = &ServiceHealth{Service: t.Service}
			healthMap[t.Service] = h
		}
		switch t.Status {
		case "running":
			h.TasksRunning++
		case "failed", "rejected":
			h.TasksFailed++
		case "pending", "assigned", "accepted", "preparing", "starting":
			h.TasksPending++
		}
	}

	for svc, h := range healthMap {
		restarts, err := s.db.ServiceRestartCount(ctx, svc, days)
		if err == nil {
			h.Restarts = restarts
		}
	}

	result := make([]ServiceHealth, 0, len(healthMap))
	for _, h := range healthMap {
		result = append(result, *h)
	}
	return result, nil
}

// buildAgentsList constrói o payload de /api/agents.
// Usado por: handleListAgents (GET) + collector/agent_handlers (SSE topic agents).
func (s *Server) buildAgentsList(ctx context.Context) ([]map[string]any, error) {
	agents, err := s.db.GetAgents(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		result = append(result, map[string]any{
			"node_id":          a.NodeID,
			"hostname":         a.Hostname,
			"version":          a.Version,
			"containers_count": a.ContainersCount,
			"last_heartbeat":   formatTimePtr(a.LastHeartbeat),
			"status":           a.Status,
			"first_seen":       formatTimePtr(a.FirstSeen),
			"updated_at":       formatTimePtr(a.UpdatedAt),
		})
	}
	return result, nil
}

// buildOOMEvents constrói o payload de /api/oom-events.
// Usado por: handleOOMEvents (GET) + collector/agent_handlers (SSE topic events).
func (s *Server) buildOOMEvents(ctx context.Context, days int, service string) ([]map[string]any, error) {
	stmt := fmt.Sprintf(
		`SELECT ts, service, container_id, exit_code FROM oom_events
		 WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days)
	if service != "" {
		stmt += " AND service = ? ORDER BY ts DESC LIMIT 200"
		rows, err := s.db.QueryContext(ctx, stmt, service)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanOOMRows(rows), nil
	}
	stmt += " ORDER BY ts DESC LIMIT 200"
	rows, err := s.db.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOOMRows(rows), nil
}

// buildChangeLog constrói o payload de /api/change-log.
// Usado por: handleChangeLog (GET) + scheduler (SSE topic change-log).
// Retorna changeLogResponse (sql.Null* desembrulhados) para o frontend.
func (s *Server) buildChangeLog(ctx context.Context, service string, limit int32) ([]changeLogResponse, error) {
	entries, err := s.db.GetChangeLog(ctx, service, limit)
	if err != nil {
		return nil, err
	}
	return toChangeLogResponses(entries), nil
}

// buildSchedulesList constrói o payload de /api/schedules.
// Usado por: handleListSchedules (GET) + scheduler (SSE topic change-log).
// Retorna scheduleResponse (sql.Null* desembrulhados) para o frontend.
func (s *Server) buildSchedulesList(ctx context.Context, status string) ([]scheduleResponse, error) {
	schedules, err := s.db.ListSchedules(ctx, status)
	if err != nil {
		return nil, err
	}
	return toScheduleResponses(schedules), nil
}

// buildRecommendations constrói o payload de /api/recommendations.
// Usado por: handleListRecommendations (GET) + collector (SSE topic services).
//
// Filtra serviços archived do service registry para manter paridade com
// buildServicesList — o ML sidecar retorna todos os serviços com métricas
// nos últimos N dias, incluindo serviços antigos/arquivados que não devem
// aparecer no Studio.
func (s *Server) buildRecommendations(ctx context.Context) (any, error) {
	result, err := s.ml.AnalyzeAll(ctx)
	if err != nil {
		return []any{}, nil
	}

	arr, ok := result.([]any)
	if !ok {
		return result, nil
	}

	registry, err := s.db.GetServiceRegistry(ctx)
	if err != nil {
		// Sem registry → retorna sem filtrar (fallback graceful)
		return arr, nil
	}

	filtered := make([]any, 0, len(arr))
	for _, item := range arr {
		rec, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		svcName, _ := rec["service"].(string)
		if reg, exists := registry[svcName]; exists && reg.Status == "archived" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

// calcRiskScore calcula um score de risco determinístico (0-100) para um serviço.
//
// Baseado em benchmarks da indústria:
//   - Kubernetes VPA: percentis + OOM ratchet
//   - Linux OOM killer: memória residente vs total
//   - Datadog: P99 vs limit + OOM bump
//
// Fórmula (pesos somam 100):
//   - Memory pressure (40 pts): mem_p99 / mem_limit
//     <70% do limite: 0 pts | 70-90%: 20 pts | >90%: 40 pts | sem limite: 15 pts
//   - OOM history (30 pts): oom_events count
//     0: 0 pts | 1-5: 15 pts | >5: 30 pts
//   - Memory leak (20 pts): ML sidecar detecta leak
//     true: 20 pts | false: 0 pts
//   - Config drift (10 pts): ML sidecar detecta drift
//     true: 10 pts | false: 0 pts
//
// Classificação:
//
//	0-25: saudável (verde) | 26-50: atenção (amarelo)
//	51-75: risco (laranja) | 76-100: crítico (vermelho)
func calcRiskScore(memP99 float64, memLimit int64, oomCount int, hasLeak, hasDrift bool) int {
	score := 0

	// 1. Memory pressure (40 pts)
	if memLimit > 0 {
		ratio := memP99 / float64(memLimit)
		switch {
		case ratio > 0.90:
			score += 40
		case ratio > 0.70:
			score += 20
		}
	} else {
		// Sem limite definido — risco de consumir toda memória do node
		score += 15
	}

	// 2. OOM history (30 pts)
	switch {
	case oomCount > 5:
		score += 30
	case oomCount > 0:
		score += 15
	}

	// 3. Memory leak (20 pts)
	if hasLeak {
		score += 20
	}

	// 4. Config drift (10 pts)
	if hasDrift {
		score += 10
	}

	if score > 100 {
		score = 100
	}
	return score
}

// BuildServiceDetailData constrói o payload completo do ServiceDetail para SSE.
// Combina stats + metrics (downsampled) + containers + tasks + health em um
// único payload, eliminando 6 GETs HTTP do frontend (zero refetch com SSE).
//
// O collector chama esta função a cada coleta para cada serviço que tem
// subscriber SSE ativo no tópico "service-detail/{name}".
//
// Payload:
//
//	{
//	  "service": "name",
//	  "stats": { ... },
//	  "metrics": [ ... ],  // downsampled ~300 pontos
//	  "containers": [ ... ],
//	  "tasks": [ ... ],
//	  "health": [ ... ]
//	}
func (s *Server) BuildServiceDetailData(ctx context.Context, service string) (map[string]any, error) {
	days := s.cfg.AnalysisWindowDays

	// 1. Stats (mesma query do handleServiceStats)
	statsQuery := fmt.Sprintf(`
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
	stats := map[string]any{"service": service, "samples": 0}
	if err := s.db.QueryRowContext(ctx, statsQuery, service).
		Scan(&samples, &cpuP50, &cpuP95, &cpuMin, &cpuMax, &cpuAvg,
			&memP50, &memP99, &memMin, &memMax, &memAvg); err == nil && samples > 0 {
		stats = map[string]any{
			"service": service,
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
		}
	}

	// 2. Metrics (downsampled, mesma lógica do handleServiceMetrics)
	metrics := s.buildServiceMetricsDownsampled(ctx, service, days)

	// 3. Containers (mesma query do handleServiceContainers)
	containers := s.buildServiceContainersData(ctx, service, days)

	// 4. Tasks (mesma lógica do handleServiceTasks)
	tasks, _ := s.db.GetTasksByService(ctx, service)
	nodes, _ := s.db.GetNodes(ctx)
	nodeHostnames := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeHostnames[n.NodeID] = n.Hostname
	}
	tasksResult := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		tasksResult = append(tasksResult, map[string]any{
			"task_id":       t.TaskID,
			"service":       t.Service,
			"node_id":       t.NodeID,
			"node_hostname": nodeHostnames[t.NodeID],
			"slot":          t.Slot,
			"status":        t.Status,
			"desired_state": t.DesiredState,
			"created_at":    formatTimePtr(t.CreatedAt),
			"updated_at":    formatTimePtr(t.UpdatedAt),
		})
	}

	// 5. Health (buildServicesHealth para todos os serviços, o frontend filtra)
	health, _ := s.buildServicesHealth(ctx, days)

	// 6. Status + risk_score + risk_factors (mesma lógica do buildServicesList,
	//    mas filtrado para um único serviço — permite que o ServiceDetail
	//    exiba o StatusBadge e RiskScoreGauge em tempo real via SSE).
	status := "legado"
	lastSeen := ""
	registry, _ := s.db.GetServiceRegistry(ctx)
	if reg, ok := registry[service]; ok {
		if reg.Status == "online" || reg.Status == "offline" || reg.Status == "legado" {
			status = reg.Status
		}
	}

	// last_seen + mem_p99 (para risk score) em 1 query
	var lastSeenSql sql.NullTime
	var memP99ForRisk float64
	_ = s.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT MAX(ts), quantile(mem_usage, 0.99)
		FROM metrics
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days), service).
		Scan(&lastSeenSql, &memP99ForRisk)
	if lastSeenSql.Valid {
		lastSeen = lastSeenSql.Time.Format(time.RFC3339Nano)
	}

	// OOM count para o serviço
	oomCount := 0
	oomRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT count(*) FROM oom_events
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days), service)
	if err == nil {
		for oomRows.Next() {
			_ = oomRows.Scan(&oomCount)
		}
		oomRows.Close()
	}

	// Leak/Drift do ML sidecar
	hasLeak, hasDrift := false, false
	if alerts, err := s.ml.GetAlerts(ctx); err == nil && alerts != nil {
		for _, a := range alerts.LeakAlerts {
			if svc, ok := a["service"].(string); ok && svc == service {
				hasLeak = true
			}
		}
		for _, a := range alerts.DriftAlerts {
			if svc, ok := a["service"].(string); ok && svc == service {
				hasDrift = true
			}
		}
	}

	// Config atual de recursos (para mem_limit do risk score)
	resourcesMap, _ := s.docker.GetAllServiceResources(ctx)
	memLimit := int64(0)
	if res, ok := resourcesMap[service]; ok {
		memLimit = res.MemLimit
	}

	riskScore := calcRiskScore(memP99ForRisk, memLimit, oomCount, hasLeak, hasDrift)

	return map[string]any{
		"service":    service,
		"stats":      stats,
		"metrics":    metrics,
		"containers": containers,
		"tasks":      tasksResult,
		"health":     health,
		"status":     status,
		"last_seen":  nullStr(lastSeen, lastSeen == ""),
		"risk_score": riskScore,
		"risk_factors": map[string]any{
			"oom_count": oomCount,
			"has_leak":  hasLeak,
			"has_drift": hasDrift,
			"mem_limit": memLimit,
		},
	}, nil
}

// buildServiceMetricsDownsampled retorna métricas temporais com downsampling.
// Se > 300 amostras, agrega em buckets temporais via time_bucket do DuckDB.
func (s *Server) buildServiceMetricsDownsampled(ctx context.Context, service string, days int) []map[string]any {
	var totalSamples int
	countQuery := fmt.Sprintf(`
		SELECT count(*) FROM metrics
		WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days)
	_ = s.db.QueryRowContext(ctx, countQuery, service).Scan(&totalSamples)

	const maxPoints = 300

	if totalSamples <= maxPoints {
		query := fmt.Sprintf(`
			SELECT ts, cpu_percent, mem_usage, mem_limit, mem_percent
			FROM metrics
			WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
			ORDER BY ts`, days)
		rows, err := s.db.QueryContext(ctx, query, service)
		if err != nil {
			return []map[string]any{}
		}
		defer rows.Close()
		return scanMetricRows(rows)
	}

	bucketSeconds := (days * 86400) / maxPoints
	if bucketSeconds < 60 {
		bucketSeconds = 60
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
	rows, err := s.db.QueryContext(ctx, query, service)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	return scanMetricRows(rows)
}

// buildContainerMetricsDownsampled retorna métricas temporais de um container
// com downsampling (mesma lógica de buildServiceMetricsDownsampled, mas
// filtrando por container_id). Usado por BuildContainerDetailData para o
// tópico "container-detail/{id}" — limita o payload SSE a ~300 pontos.
func (s *Server) buildContainerMetricsDownsampled(ctx context.Context, containerID string, days int) []map[string]any {
	var totalSamples int
	countQuery := fmt.Sprintf(`
		SELECT count(*) FROM metrics
		WHERE container_id = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS`, days)
	_ = s.db.QueryRowContext(ctx, countQuery, containerID).Scan(&totalSamples)

	const maxPoints = 300

	if totalSamples <= maxPoints {
		query := fmt.Sprintf(`
			SELECT ts, cpu_percent, mem_usage, mem_limit, mem_percent
			FROM metrics
			WHERE container_id = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
			ORDER BY ts`, days)
		rows, err := s.db.QueryContext(ctx, query, containerID)
		if err != nil {
			return []map[string]any{}
		}
		defer rows.Close()
		return scanMetricRows(rows)
	}

	bucketSeconds := (days * 86400) / maxPoints
	if bucketSeconds < 60 {
		bucketSeconds = 60
	}
	query := fmt.Sprintf(`
		SELECT
			time_bucket(INTERVAL '%d seconds', ts) as bucket_ts,
			avg(cpu_percent) as cpu_percent,
			avg(mem_usage) as mem_usage,
			max(mem_limit) as mem_limit,
			avg(mem_percent) as mem_percent
		FROM metrics
		WHERE container_id = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY bucket_ts
		ORDER BY bucket_ts`, bucketSeconds, days)
	rows, err := s.db.QueryContext(ctx, query, containerID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	return scanMetricRows(rows)
}

// BuildContainerDetailData constrói o payload completo do ContainerDetail para SSE.
// Combina stats + metrics (downsampled) + network em um único payload,
// eliminando 3 GETs HTTP do frontend (zero refetch com SSE).
//
// O collector chama esta função a cada coleta para cada container que tem
// subscriber SSE ativo no tópico "container-detail/{id}".
//
// Payload:
//
//	{
//	  "container_id": "id",
//	  "stats": { ... },          // mesmo shape do handleContainerStats
//	  "metrics": [ ... ],        // downsampled ~300 pontos (mesmo shape do handleContainerMetrics)
//	  "network": [ ... ]         // mesmo shape do handleContainerNetworkInfo
//	}
func (s *Server) BuildContainerDetailData(ctx context.Context, containerID string) (map[string]any, error) {
	days := s.cfg.AnalysisWindowDays

	// 1. Stats (mesma query do handleContainerStats)
	statsQuery := fmt.Sprintf(`
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
	stats := map[string]any{"container_id": containerID, "samples": 0}
	if err := s.db.QueryRowContext(ctx, statsQuery, containerID).
		Scan(&samples, &cpuP50, &cpuP95, &cpuMin, &cpuMax, &cpuAvg,
			&memP50, &memP99, &memMin, &memMax, &memAvg); err == nil && samples > 0 {
		stats = map[string]any{
			"container_id": containerID,
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
		}
	}

	// 2. Metrics (downsampled, mesmo shape do handleContainerMetrics)
	metrics := s.buildContainerMetricsDownsampled(ctx, containerID, days)

	// 3. Network (mesmo shape do handleContainerNetworkInfo)
	networks, err := s.docker.GetContainerNetworks(ctx, containerID)
	if err != nil {
		networks = []docker.ContainerNetwork{}
	}
	if networks == nil {
		networks = []docker.ContainerNetwork{}
	}

	return map[string]any{
		"container_id": containerID,
		"stats":        stats,
		"metrics":      metrics,
		"network":      networks,
	}, nil
}

func scanMetricRows(rows *sql.Rows) []map[string]any {
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
	return result
}

// buildServiceContainersData retorna containers de um serviço com stats.
func (s *Server) buildServiceContainersData(ctx context.Context, service string, days int) []map[string]any {
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
	rows, err := s.db.QueryContext(ctx, query, service)
	if err != nil {
		return []map[string]any{}
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
		}
		if lastSeen.Valid && lastSeen.Time.After(fiveMinAgo) && !runningIDs[cid] {
			status = "offline"
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
			"last_seen":    formatTimePtr(&lastSeen.Time),
			"status":       status,
			"networks":     containerNetworksMap[cid],
		})
	}
	return result
}
