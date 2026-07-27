// Package server — /api/nodes/* handlers.
//
// Portado de backend/routers/nodes.py.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/resma/api/internal/db"
)

// registerNodeRoutes registra as rotas de nodes no mux interno.
func (s *Server) registerNodeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/nodes", s.handleListNodes)
	mux.HandleFunc("GET /api/nodes/cluster", s.handleClusterInfo)
	mux.HandleFunc("GET /api/nodes/{node_id}", s.handleNodeDetail)
	mux.HandleFunc("GET /api/nodes/{node_id}/metrics", s.handleNodeMetrics)
	mux.HandleFunc("GET /api/nodes/{node_id}/services", s.handleNodeServices)
}

// handleListNodes lista todos os nós com consumo agregado.
func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	result, err := s.buildNodesList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// handleClusterInfo retorna info do cluster Swarm.
func (s *Server) handleClusterInfo(w http.ResponseWriter, r *http.Request) {
	result, err := s.buildClusterInfo(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// handleNodeDetail retorna detalhes de um nó.
func (s *Server) handleNodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := pathValue(r, "node_id")
	node, err := s.db.GetNodeByID(r.Context(), nodeID)
	if err != nil || node == nil {
		writeError(w, http.StatusNotFound, "Node not found")
		return
	}
	consumption, _ := s.db.GetNodeConsumption(r.Context(), nodeID, s.cfg.AnalysisWindowDays)
	labels := map[string]any{}
	if node.Labels != "" {
		_ = json.Unmarshal([]byte(node.Labels), &labels)
	}
	writeOK(w, map[string]any{
		"node_id":        node.NodeID,
		"hostname":       node.Hostname,
		"role":           node.Role,
		"availability":   node.Availability,
		"status":         node.Status,
		"address":        node.Address,
		"cpu_total":      node.CPUTotal,
		"mem_total":      node.MemTotal,
		"os":             node.OS,
		"architecture":   node.Architecture,
		"engine_version": node.EngineVersion,
		"is_leader":      node.IsLeader,
		"reachability":   node.Reachability,
		"labels":         labels,
		"tasks_running":  node.TasksRunning,
		"cpu_p95":        consumption.CPUP95,
		"mem_p99":        consumption.MemP99,
		"containers":     consumption.Containers,
		"updated_at":     formatTime(node.UpdatedAt),
	})
}

// handleNodeMetrics retorna métricas temporais de um nó.
func (s *Server) handleNodeMetrics(w http.ResponseWriter, r *http.Request) {
	nodeID := pathValue(r, "node_id")
	metrics, err := s.db.GetNodeMetrics(r.Context(), nodeID, s.cfg.AnalysisWindowDays)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if metrics == nil {
		metrics = []db.NodeMetricPoint{}
	}
	writeOK(w, metrics)
}

// handleNodeServices retorna serviços rodando em um nó.
func (s *Server) handleNodeServices(w http.ResponseWriter, r *http.Request) {
	nodeID := pathValue(r, "node_id")
	services, err := s.db.GetNodeServices(r.Context(), nodeID, s.cfg.AnalysisWindowDays)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if services == nil {
		services = []db.NodeService{}
	}
	writeOK(w, services)
}

// formatTime formata time.Time para RFC3339 ou nil se zero.
func formatTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}
