// Package server — handlers para ingestão de dados dos agents (Fase 7).
//
// Estes endpoints (/api/agent/*) recebem métricas, OOM events e heartbeats
// dos RESMA Agents rodando em cada node do Swarm. A autenticação é via
// bearer token compartilhado (RESMA_AGENT_TOKEN) — não JWT, não API key.
//
// Rotas:
//
//	POST /api/agent/ingest/metrics  — batch de MetricPoint
//	POST /api/agent/ingest/oom      — OOMEvent
//	POST /api/agent/heartbeat       — Heartbeat
//
// Resiliência: o agent tem ring buffer + persistência em disco, então o
// server pode rejeitar (429/5xx) que o agent retentará com backoff.
package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/db"
)

// registerAgentRoutes registra as rotas /api/agent/* no mux informado.
// O mux já deve estar envolvido pelo agentTokenMiddleware (registrado em
// server.go no grupo /api/agent/).
func (s *Server) registerAgentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/agent/ingest/metrics", s.handleAgentIngestMetrics)
	mux.HandleFunc("POST /api/agent/ingest/oom", s.handleAgentIngestOOM)
	mux.HandleFunc("POST /api/agent/heartbeat", s.handleAgentHeartbeat)
}

// agentTokenMiddleware valida o bearer token compartilhado (RESMA_AGENT_TOKEN).
// Rejeita com 401 se o token estiver vazio na config ou não bater.
// Usa constant-time comparison para evitar timing attacks.
func (s *Server) agentTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := s.cfg.AgentToken
		if expected == "" {
			// Se o token não está configurado, rejeitar tudo (fail-closed).
			writeError(w, http.StatusUnauthorized, "agent token not configured on server")
			return
		}
		token := extractAgentBearer(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing agent token")
			return
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid agent token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractAgentBearer extrai o token do header "Authorization: Bearer <token>".
func extractAgentBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// agentMetricsBatch é o payload de POST /api/agent/ingest/metrics.
type agentMetricsBatch struct {
	NodeID  string             `json:"node_id"`
	Metrics []agentMetricPoint `json:"metrics"`
}

// agentMetricPoint é um ponto de métrica enviado pelo agent.
// Mantém os mesmos campos do db.MetricRow para 1:1 mapping.
type agentMetricPoint struct {
	TS                  string  `json:"ts"`
	Service             string  `json:"service"`
	ContainerID         string  `json:"container_id"`
	CPUPercent          float64 `json:"cpu_percent"`
	CPUUsage            int64   `json:"cpu_usage"`
	CPUSystem           int64   `json:"cpu_system"`
	MemUsage            int64   `json:"mem_usage"`
	MemLimit            int64   `json:"mem_limit"`
	MemPercent          float64 `json:"mem_percent"`
	NetRX               int64   `json:"net_rx"`
	NetTX               int64   `json:"net_tx"`
	BlockRead           int64   `json:"block_read"`
	BlockWrite          int64   `json:"block_write"`
	MemWorkingSet       int64   `json:"mem_working_set"`
	CPUThrottledPeriods int64   `json:"cpu_throttled_periods"`
	CPUThrottledTime    int64   `json:"cpu_throttled_time"`
}

// handleAgentIngestMetrics recebe um batch de métricas de um agent.
func (s *Server) handleAgentIngestMetrics(w http.ResponseWriter, r *http.Request) {
	var batch agentMetricsBatch
	if !decodeJSON(w, r, &batch) {
		return
	}
	if batch.NodeID == "" {
		writeError(w, http.StatusBadRequest, "missing node_id")
		return
	}
	if len(batch.Metrics) == 0 {
		writeJSON(w, http.StatusOK, map[string]int{"accepted": 0})
		return
	}

	rows := make([]db.MetricRow, 0, len(batch.Metrics))
	for _, m := range batch.Metrics {
		ts, err := time.Parse(time.RFC3339Nano, m.TS)
		if err != nil {
			// fallback: usar now() se o timestamp for inválido
			ts = time.Now().UTC()
		}
		rows = append(rows, db.MetricRow{
			TS:                  ts,
			Service:             m.Service,
			ContainerID:         m.ContainerID,
			CPUPercent:          m.CPUPercent,
			CPUUsage:            m.CPUUsage,
			CPUSystem:           m.CPUSystem,
			MemUsage:            m.MemUsage,
			MemLimit:            m.MemLimit,
			MemPercent:          m.MemPercent,
			NetRX:               m.NetRX,
			NetTX:               m.NetTX,
			BlockRead:           m.BlockRead,
			BlockWrite:          m.BlockWrite,
			MemWorkingSet:       m.MemWorkingSet,
			CPUThrottledPeriods: m.CPUThrottledPeriods,
			CPUThrottledTime:    m.CPUThrottledTime,
			NodeID:              batch.NodeID, // Fase 7 — origem da métrica
		})
	}

	if err := s.db.InsertMetricsBatch(r.Context(), rows); err != nil {
		s.log.Error("agent ingest metrics falhou", "node_id", batch.NodeID, "count", len(rows), "err", err)
		writeError(w, http.StatusInternalServerError, "insert failed")
		return
	}

	// Atualizar container_node_map para correlação container → node (batch)
	mapRows := make([]db.ContainerNodeMapRow, 0, len(batch.Metrics))
	seen := make(map[string]bool, len(batch.Metrics))
	for _, m := range batch.Metrics {
		if m.ContainerID != "" && !seen[m.ContainerID] {
			seen[m.ContainerID] = true
			mapRows = append(mapRows, db.ContainerNodeMapRow{
				ContainerID: m.ContainerID,
				NodeID:      batch.NodeID,
				Service:     m.Service,
			})
		}
	}
	if len(mapRows) > 0 {
		_ = s.db.UpsertContainerNodeMapBatch(r.Context(), mapRows)
	}

	// NÃO publicar SSE aqui. Os agents fazem ingest em batch (~10-20 POSTs/s
	// com 4 nodes), e publicar SSE a cada ingest causaria:
	// - 10-20 publicações/s do tópico "metrics" (cada uma chamando
	//   buildDashboardData — query pesada)
	// - 10-20 setQueryData(["dashboard"]) no frontend por segundo
	// - Picos de CPU/memória no api e no browser
	//
	// O collector (collectLoop) já publica SSE a cada CollectInterval (5s)
	// com o payload completo do dashboard. A ingest de agent só precisa
	// inserir no DB — o collector cuida da notificação SSE.

	writeJSON(w, http.StatusOK, map[string]int{"accepted": len(rows)})
}

// agentOOMEvent é o payload de POST /api/agent/ingest/oom.
type agentOOMEvent struct {
	TS          string `json:"ts"`
	Service     string `json:"service"`
	ContainerID string `json:"container_id"`
	ExitCode    int    `json:"exit_code"`
}

// handleAgentIngestOOM recebe um evento OOM de um agent.
func (s *Server) handleAgentIngestOOM(w http.ResponseWriter, r *http.Request) {
	var ev agentOOMEvent
	if !decodeJSON(w, r, &ev) {
		return
	}
	// node_id vem do header X-RESMA-Node-ID (setado pelo agent) ou query
	nodeID := r.Header.Get("X-RESMA-Node-ID")
	if nodeID == "" {
		nodeID = r.URL.Query().Get("node_id")
	}

	ts, err := time.Parse(time.RFC3339Nano, ev.TS)
	if err != nil {
		ts = time.Now().UTC()
	}

	if err := s.db.InsertOOMEventWithNode(r.Context(), ts, ev.Service, ev.ContainerID, nodeID, int32(ev.ExitCode)); err != nil {
		s.log.Error("agent ingest oom falhou", "service", ev.Service, "node_id", nodeID, "err", err)
		writeError(w, http.StatusInternalServerError, "insert failed")
		return
	}

	// Notificar SSE (tópico events) — payload completo de OOM events.
	// O frontend usa setQueryData(["oom-events"], payload) — zero refetch.
	if s.sse != nil {
		if data, err := s.buildOOMEvents(r.Context(), 7, ""); err == nil && data != nil {
			s.sse.Publish("events", "oom", data)
		}
	}

	s.log.Warn("OOM event recebido do agent", "service", ev.Service, "node_id", nodeID, "exit_code", ev.ExitCode)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

// agentHeartbeat é o payload de POST /api/agent/heartbeat.
type agentHeartbeat struct {
	NodeID          string `json:"node_id"`
	Hostname        string `json:"hostname"`
	ContainersCount int    `json:"containers_count"`
	Version         string `json:"version"`
}

// handleAgentHeartbeat recebe um heartbeat de um agent e faz upsert na tabela agents.
func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	var hb agentHeartbeat
	if !decodeJSON(w, r, &hb) {
		return
	}
	if hb.NodeID == "" {
		writeError(w, http.StatusBadRequest, "missing node_id")
		return
	}

	if err := s.db.UpsertAgent(r.Context(), hb.NodeID, hb.Hostname, hb.Version, int32(hb.ContainersCount)); err != nil {
		s.log.Error("agent heartbeat upsert falhou", "node_id", hb.NodeID, "err", err)
		writeError(w, http.StatusInternalServerError, "upsert failed")
		return
	}

	// Notificar SSE (tópico agents) — payload completo de agents.
	// O frontend usa setQueryData(["agents"], payload) — zero refetch.
	if s.sse != nil {
		if data, err := s.buildAgentsList(r.Context()); err == nil && data != nil {
			s.sse.Publish("agents", "heartbeat", data)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
