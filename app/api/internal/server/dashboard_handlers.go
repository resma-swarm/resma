// Package server — /api/dashboard handler.
//
// Portado de backend/routers/dashboard.py. Alertas detalhados (OOMs, leaks,
// drifts) foram movidos para /api/alerts (ver alert_handlers.go). O dashboard
// mantém apenas um resumo de contagem (alerts_summary) para o badge com link.
package server

import (
	"encoding/json"
	"net/http"
)

// handleDashboard retorna o blob agregado do dashboard UI.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildDashboardData(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, data)
}

// parseJSONWarnings faz parse de warnings JSON string para []any.
func parseJSONWarnings(warnings string) []any {
	if warnings == "" {
		return []any{}
	}
	var out []any
	_ = json.Unmarshal([]byte(warnings), &out)
	if out == nil {
		out = []any{}
	}
	return out
}
