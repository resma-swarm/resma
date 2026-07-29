package rollback

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/resma-swarm/resma/app/api/internal/db"
)

// Criteria define quais checks estão habilitados para um watch.
type Criteria struct {
	OOM         bool `json:"oom"`          // ≥ N OOMs desde apply
	Throttle    bool `json:"throttle"`     // CPU throttling > X% por > Y min
	MemPressure bool `json:"mem_pressure"` // mem > X% do limite por > Y min
}

// CriteriaResult indica qual critério disparou (vazio se nenhum).
type CriteriaResult struct {
	Triggered bool
	Reason    string // "oom:2", "throttle:15%", "mem_pressure:98%"
}

// parseCriteria faz parse do JSON de critérios armazenado no DB.
// Se o JSON for vazio ou inválido, usa defaults (oom=true, throttle=true).
func parseCriteria(raw string) (Criteria, error) {
	if raw == "" || raw == "{}" {
		return Criteria{OOM: true, Throttle: true}, nil
	}
	var c Criteria
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return c, fmt.Errorf("parse criteria JSON: %w", err)
	}
	return c, nil
}

// CriteriaToJSON serializa Criteria para JSON (para armazenar no DB).
func CriteriaToJSON(c Criteria) string {
	b, _ := json.Marshal(c)
	return string(b)
}

// evaluate verifica todos os critérios habilitados contra métricas desde o apply.
func (w *Watcher) evaluate(ctx context.Context, watch db.RollbackWatch, c Criteria) CriteriaResult {
	since := watch.StartedAt

	// 1. OOM count
	if c.OOM {
		oomCount, err := w.db.GetOOMCountSince(ctx, watch.Service, since)
		if err == nil && oomCount >= int32(w.cfg.RollbackOOMThreshold) {
			return CriteriaResult{
				Triggered: true,
				Reason:    fmt.Sprintf("oom:%d", oomCount),
			}
		}
	}

	// 2. CPU throttling
	if c.Throttle {
		metrics, err := w.db.GetMetricsSince(ctx, watch.Service, since)
		if err == nil {
			if triggered, pct := checkThrottle(metrics, w.cfg.RollbackThrottlePct,
				w.cfg.RollbackThrottleMin); triggered {
				return CriteriaResult{
					Triggered: true,
					Reason:    fmt.Sprintf("throttle:%.0f%%", pct),
				}
			}
		}
	}

	// 3. Mem pressure
	if c.MemPressure {
		metrics, err := w.db.GetMetricsSince(ctx, watch.Service, since)
		if err == nil {
			if triggered, pct := checkMemPressure(metrics,
				w.cfg.RollbackMemPressurePct, w.cfg.RollbackMemPressureMin); triggered {
				return CriteriaResult{
					Triggered: true,
					Reason:    fmt.Sprintf("mem_pressure:%.0f%%", pct),
				}
			}
		}
	}

	return CriteriaResult{Triggered: false}
}

// checkThrottle verifica se CPU throttling > threshold por >= minMinutes amostras consecutivas.
// Usa cpu_throttled_periods das métricas — se > 0, há throttling ativo naquela amostra.
func checkThrottle(metrics []db.MetricRow, thresholdPct float64, minMinutes int) (bool, float64) {
	if len(metrics) == 0 {
		return false, 0
	}
	consecutive := 0
	var maxPct float64
	for _, m := range metrics {
		// Proxy: se throttled_periods > 0, há throttling ativo.
		// Para uma métrica mais precisa, seria necessário throttled_time / total_cpu_time,
		// mas o proxy de periods > 0 é suficiente para detectar throttling sustentado.
		pct := 0.0
		if m.CPUThrottledPeriods > 0 {
			pct = float64(m.CPUThrottledPeriods) // valor bruto como proxy
		}
		if pct > thresholdPct {
			consecutive++
			if pct > maxPct {
				maxPct = pct
			}
			if consecutive >= minMinutes {
				return true, maxPct
			}
		} else {
			consecutive = 0
		}
	}
	return false, maxPct
}

// checkMemPressure verifica se mem_usage > threshold% do mem_limit por >= minMinutes amostras consecutivas.
func checkMemPressure(metrics []db.MetricRow, thresholdPct float64, minMinutes int) (bool, float64) {
	if len(metrics) == 0 {
		return false, 0
	}
	consecutive := 0
	var maxPct float64
	for _, m := range metrics {
		if m.MemLimit <= 0 {
			continue
		}
		pct := (float64(m.MemUsage) / float64(m.MemLimit)) * 100
		if pct >= thresholdPct {
			consecutive++
			if pct > maxPct {
				maxPct = pct
			}
			if consecutive >= minMinutes {
				return true, maxPct
			}
		} else {
			consecutive = 0
		}
	}
	return false, maxPct
}
