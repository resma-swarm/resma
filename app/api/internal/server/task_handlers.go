// Package server — /api/agents/* e /api/tasks/* handlers (Fase 7 — admin).
//
// Estes endpoints expõem o estado dos RESMA Agents e do task lifecycle do
// Swarm para o frontend (JWT auth, rotas internas /api/*).
//
// Rotas:
//
//	GET /api/agents              — lista todos os agents com status
//	GET /api/agents/{node_id}    — detalhe de um agent
//	GET /api/tasks               — lista todas as tasks (com node hostname via LEFT JOIN)
//	GET /api/tasks/{service}     — tasks de um serviço específico
//	GET /api/tasks/{service}/history — histórico de mudanças de status (restarts)
//	GET /api/services/health     — health de todos os serviços (restarts, tasks running/failed)
package server

import (
	"net/http"
	"time"

	"github.com/resma/api/internal/db"
)

// registerAgentAdminRoutes registra as rotas admin de agents no mux interno.
// (Não confundir com registerAgentRoutes — esse é para ingestão, sem JWT.)
func (s *Server) registerAgentAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/agents/{node_id}", s.handleAgentDetail)
}

// registerTaskRoutes registra as rotas de tasks no mux interno.
func (s *Server) registerTaskRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	mux.HandleFunc("GET /api/tasks/{service}", s.handleServiceTasks)
	mux.HandleFunc("GET /api/tasks/{service}/history", s.handleTaskHistory)
	mux.HandleFunc("GET /api/services/health", s.handleServicesHealth)
}

// handleListAgents lista todos os agents com status (active/stale).
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	result, err := s.buildAgentsList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// handleAgentDetail retorna o detalhe de um agent pelo node_id.
func (s *Server) handleAgentDetail(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing node_id")
		return
	}
	agent, err := s.db.GetAgentByNode(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeOK(w, map[string]any{
		"node_id":          agent.NodeID,
		"hostname":         agent.Hostname,
		"version":          agent.Version,
		"containers_count": agent.ContainersCount,
		"last_heartbeat":   formatTimePtr(agent.LastHeartbeat),
		"status":           agent.Status,
		"first_seen":       formatTimePtr(agent.FirstSeen),
		"updated_at":       formatTimePtr(agent.UpdatedAt),
	})
}

// handleListTasks lista todas as tasks com hostname do node (LEFT JOIN nodes).
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	result, err := s.buildTasksList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// handleServiceTasks retorna as tasks de um serviço específico.
func (s *Server) handleServiceTasks(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "missing service")
		return
	}
	tasks, err := s.db.GetTasksByService(r.Context(), service)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Buscar hostnames dos nodes
	nodes, _ := s.db.GetNodes(r.Context())
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
	writeOK(w, result)
}

// handleTaskHistory retorna o histórico de mudanças de status de um serviço.
func (s *Server) handleTaskHistory(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "missing service")
		return
	}
	days := parseDaysQuery(r, 7)
	history, err := s.db.GetTaskHistory(r.Context(), service, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]map[string]any, 0, len(history))
	for _, h := range history {
		result = append(result, map[string]any{
			"ts":      h.TS.Format(time.RFC3339Nano),
			"task_id": h.TaskID,
			"status":  h.Status,
			"slot":    h.Slot,
			"node_id": h.NodeID,
		})
	}
	writeOK(w, result)
}

// ServiceHealth é o health de um serviço (para o card de service health).
type ServiceHealth struct {
	Service       string  `json:"service"`
	TasksRunning  int32   `json:"tasks_running"`
	TasksFailed   int32   `json:"tasks_failed"`
	TasksPending  int32   `json:"tasks_pending"`
	Restarts      int32   `json:"restarts"`
	LastRestartAt *string `json:"last_restart_at"`
}

// handleServicesHealth retorna o health de todos os serviços com tasks no Swarm.
// Agrega: tasks running/failed/pending, restarts (transições para running nos últimos N dias).
func (s *Server) handleServicesHealth(w http.ResponseWriter, r *http.Request) {
	days := parseDaysQuery(r, 7)
	result, err := s.buildServicesHealth(r.Context(), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, result)
}

// formatTimePtr formata *time.Time para RFC3339 ou nil se nil ou zero.
func formatTimePtr(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

// garantir que db é referenciado (usado em handleServicesHealth via GetTasks).
var _ = db.Task{}
