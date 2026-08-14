// Package server — /api/alerts handler.
//
// Agrega em um único feed normalizado os alertas de 3 fontes:
//   - OOMs: eventos factuais persistidos em oom_events (DuckDB)
//   - Memory leaks: inferência estatística do ML sidecar (regressão linear)
//   - Resource drifts: inferência estatística do ML sidecar (delta P50/P95)
//
// Leaks e drifts são estado derivado das métricas (computados on-demand pelo
// ML sidecar) e NÃO são persistidos — se a condição parar, o alerta some na
// próxima chamada. OOMs são eventos discretos e já estão persistidos.
//
// Shape normalizado por item:
//
//	{
//	  "type":      "oom" | "leak" | "drift",
//	  "severity":  "critical" | "warning",
//	  "service":   string,
//	  "message":   string,
//	  "details":   { ... campos específicos do tipo ... },
//	  "ts":        string (RFC3339Nano, ou vazio para alertas stateless)
//	}
//
// Compatível com futura integração Grafana/Alertmanager: o RESMA fica no
// básico (detecção + listagem); routing/silencing/notifications ficam externos.
package server

import (
	"fmt"
	"net/http"
	"time"
)

// AlertItem é o shape normalizado de um alerta no feed /api/alerts.
type AlertItem struct {
	Type     string         `json:"type"`
	Severity string         `json:"severity"`
	Service  string         `json:"service"`
	Message  string         `json:"message"`
	Details  map[string]any `json:"details"`
	TS       string         `json:"ts"`
}

// handleAlerts agrega OOMs (DuckDB) + leaks/drifts (ML sidecar) num único feed.
// OOMs são limitados pela janela de análise (cfg.AnalysisWindowDays).
// Leaks/drifts vêm do ML sidecar com fallback graceful (array vazio) se
// o sidecar estiver indisponível.
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := s.cfg.AnalysisWindowDays

	alerts := []AlertItem{}

	// --- OOMs (DuckDB) — eventos factuais persistidos ---
	oomQuery := fmt.Sprintf(`
		SELECT ts, service, count(*) as oom_count
		FROM oom_events
		WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY ts, service
		ORDER BY ts DESC
		LIMIT 50`, days)
	oomRows, err := s.db.QueryContext(ctx, oomQuery)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for oomRows.Next() {
		var ts time.Time
		var svc string
		var oomCount int32
		if err := oomRows.Scan(&ts, &svc, &oomCount); err != nil {
			continue
		}
		alerts = append(alerts, AlertItem{
			Type:     "oom",
			Severity: "critical",
			Service:  svc,
			Message:  fmt.Sprintf("%d OOM(s) detectado(s) — container encerrado pelo kernel (exit 137)", oomCount),
			Details: map[string]any{
				"oom_count": oomCount,
			},
			TS: ts.Format(time.RFC3339Nano),
		})
	}
	_ = oomRows.Close()

	// --- Leaks e Drifts (ML sidecar) — estado derivado on-demand ---
	// Fallback graceful: se o sidecar estiver indisponível, retornamos apenas
	// os OOMs (já coletados) sem falhar a página inteira.
	// now é usado como timestamp para alertas stateless (leaks/drifts) —
	// representam quando o alerta foi detectado nesta avaliação on-demand.
	now := time.Now().UTC()

	if mlAlerts, err := s.ml.GetAlerts(ctx); err == nil && mlAlerts != nil {
		for _, a := range mlAlerts.LeakAlerts {
			svc, _ := a["service"].(string)
			growth, _ := toFloat(a["daily_growth_mb"])
			r2, _ := toFloat(a["r_squared"])
			alerts = append(alerts, AlertItem{
				Type:     "leak",
				Severity: "warning",
				Service:  svc,
				Message:  fmt.Sprintf("Memory leak: crescimento de %.2f MB/dia (R²=%.3f)", growth, r2),
				Details: map[string]any{
					"daily_growth_mb": growth,
					"r_squared":       r2,
				},
				TS: now.Format(time.RFC3339Nano),
			})
		}
		for _, a := range mlAlerts.DriftAlerts {
			svc, _ := a["service"].(string)
			cpuDrift, _ := toFloat(a["cpu_drift"])
			memDrift, _ := toFloat(a["mem_drift"])
			alerts = append(alerts, AlertItem{
				Type:     "drift",
				Severity: "warning",
				Service:  svc,
				Message:  fmt.Sprintf("Resource drift: CPU %.0f%%, Mem %.0f%%", cpuDrift*100, memDrift*100),
				Details: map[string]any{
					"cpu_drift": cpuDrift,
					"mem_drift": memDrift,
				},
				TS: now.Format(time.RFC3339Nano),
			})
		}
	}

	// Ordenação: critical primeiro, depois por ts DESC (OOMs) — stateless
	// (leaks/drifts) ficam por último dentro de cada severidade.
	sortAlerts(alerts)

	writeOK(w, map[string]any{
		"alerts": alerts,
		"counts": map[string]int{
			"total":    len(alerts),
			"oom":      countByType(alerts, "oom"),
			"leak":     countByType(alerts, "leak"),
			"drift":    countByType(alerts, "drift"),
			"critical": countBySeverity(alerts, "critical"),
			"warning":  countBySeverity(alerts, "warning"),
		},
	})
}

// sortAlerts ordena por severidade (critical > warning) e depois por ts DESC.
// Alertas stateless (ts vazio) ficam após os datados dentro da mesma severidade.
func sortAlerts(alerts []AlertItem) {
	for i := 1; i < len(alerts); i++ {
		for j := i; j > 0 && alertLess(alerts[j], alerts[j-1]); j-- {
			alerts[j], alerts[j-1] = alerts[j-1], alerts[j]
		}
	}
}

// alertLess retorna true se a deve vir antes de b.
// Ordem: critical antes de warning; mesmo severity, ts DESC (vazio por último).
func alertLess(a, b AlertItem) bool {
	sa := severityRank(a.Severity)
	sb := severityRank(b.Severity)
	if sa != sb {
		return sa < sb
	}
	// Mesma severidade: ts DESC (mais recente primeiro); vazio vai pro fim.
	if a.TS == b.TS {
		return false
	}
	if a.TS == "" {
		return false
	}
	if b.TS == "" {
		return true
	}
	return a.TS > b.TS
}

func severityRank(s string) int {
	if s == "critical" {
		return 0
	}
	return 1
}

func countByType(alerts []AlertItem, t string) int {
	c := 0
	for _, a := range alerts {
		if a.Type == t {
			c++
		}
	}
	return c
}

func countBySeverity(alerts []AlertItem, s string) int {
	c := 0
	for _, a := range alerts {
		if a.Severity == s {
			c++
		}
	}
	return c
}

// toFloat converte any (pode vir como float64 ou int do JSON) para float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
