// Package server — /health e /ready endpoints.
package server

import (
	"net/http"
)

// handleHealth retorna 200 OK se o processo está vivo.
// Sem dependências — usado por load balancers / Docker healthcheck.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{"status": "ok"})
}

// handleReady verifica dependências (DuckDB, Docker) e retorna 200 ou 503.
// Usado por Kubernetes/Swarm readiness probes.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	deps := make(map[string]string)
	allOK := true

	// DuckDB
	if s.db == nil {
		deps["db"] = "not initialized"
		allOK = false
	} else {
		deps["db"] = "ok"
	}

	// Docker
	if s.docker == nil {
		deps["docker"] = "not initialized"
		allOK = false
	} else if err := s.docker.Health(r.Context()); err != nil {
		deps["docker"] = err.Error()
		allOK = false
	} else {
		deps["docker"] = "ok"
	}

	status := "ok"
	if !allOK {
		status = "degraded"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": status,
		"deps":   deps,
	})
}
