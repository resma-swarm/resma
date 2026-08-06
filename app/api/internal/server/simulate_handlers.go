// Package server — /api/recommendations/simulate
// (Right-Sizing Studio R6 — simulação em lote).
package server

import (
	"net/http"
	"time"
)

// registerSimulateRoutes registra as rotas de simulate.
func (s *Server) registerSimulateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/recommendations/simulate", s.handleSimulateBatch)
}

// handleSimulateBatch calcula o delta de recursos para um conjunto de serviços
// sob um tier específico. Não modifica estado (read-only).
//
// Spec: api-contract.md §1 POST /api/recommendations/simulate
func (s *Server) handleSimulateBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Services []string `json:"services"`
		Tier     string   `json:"tier"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Services) == 0 {
		writeError(w, http.StatusBadRequest, `{"error":"empty_services","message":"services array must not be empty"}`)
		return
	}
	tier := req.Tier
	if tier == "" {
		tier = "balanced"
	}
	if tier != "conservative" && tier != "balanced" && tier != "aggressive" {
		writeError(w, http.StatusBadRequest, `{"error":"invalid_tier","message":"tier must be one of: conservative, balanced, aggressive"}`)
		return
	}

	ctx := r.Context()
	// Buscar todas as recomendações do ML sidecar
	recsRaw, err := s.ml.AnalyzeAll(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "ML sidecar indisponível")
		return
	}
	recs, ok := recsRaw.([]any)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "ML sidecar retornou formato inválido")
		return
	}

	// Indexar recomendações por service name
	recByService := make(map[string]map[string]any, len(recs))
	for _, r := range recs {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if name, ok := rm["service"].(string); ok {
			recByService[name] = rm
		}
	}

	// Para cada serviço solicitado, calcular delta
	rows := make([]map[string]any, 0, len(req.Services))
	var totalCPUCores, totalMemBytes float64
	validCount := 0

	for _, svcName := range req.Services {
		rec, ok := recByService[svcName]
		if !ok {
			rows = append(rows, map[string]any{
				"service":         svcName,
				"current":         nil,
				"suggested":       nil,
				"resources_freed": nil,
				"risk":            nil,
				"error":           "service_not_found",
			})
			continue
		}

		// Buscar current do Docker
		current, _ := s.docker.GetServiceResources(ctx, svcName)

		// Buscar suggested_tiers[tier]
		tiers, _ := rec["suggested_tiers"].(map[string]any)
		tierData, _ := tiers[tier].(map[string]any)
		if tierData == nil {
			// Fallback: usar suggested direto
			tierData = map[string]any{
				"cpu_limit":       getFloat(rec, "suggested", "cpu_limit"),
				"mem_limit":       getFloat(rec, "suggested", "mem_limit"),
				"cpu_reservation": getFloat(rec, "suggested", "cpu_reservation"),
				"mem_reservation": getFloat(rec, "suggested", "mem_reservation"),
			}
		}

		suggestedCPU := getFloatVal(tierData["cpu_limit"])
		suggestedMem := getFloatVal(tierData["mem_limit"])

		// resources_freed = current - suggested (nunca negativo)
		var freedCPU, freedMem float64
		if current.CPULimit > suggestedCPU {
			freedCPU = current.CPULimit - suggestedCPU
		}
		if float64(current.MemLimit) > suggestedMem {
			freedMem = float64(current.MemLimit) - suggestedMem
		}

		totalCPUCores += freedCPU
		totalMemBytes += freedMem
		validCount++

		// Risk do payload ML (se disponível)
		var risk any
		if r, ok := rec["risk"].(map[string]any); ok {
			risk = r
		}

		rows = append(rows, map[string]any{
			"service":   svcName,
			"current":   map[string]any{"cpu_cores": current.CPULimit, "mem_bytes": current.MemLimit},
			"suggested": map[string]any{"cpu_cores": suggestedCPU, "mem_bytes": suggestedMem},
			"resources_freed": map[string]any{
				"cpu_cores": freedCPU,
				"mem_bytes": freedMem,
			},
			"risk": risk,
		})
	}

	if validCount == 0 {
		writeError(w, http.StatusUnprocessableEntity, `{"error":"no_valid_services","message":"nenhum dos serviços existe ou tem recomendação"}`)
		return
	}

	writeOK(w, map[string]any{
		"tier":         tier,
		"simulated_at": time.Now().UTC().Format(time.RFC3339),
		"rows":         rows,
		"totals": map[string]any{
			"services":        validCount,
			"cpu_cores_freed": totalCPUCores,
			"mem_bytes_freed": totalMemBytes,
		},
	})
}

// getFloat navega em map aninhado e retorna float64 do último key.
func getFloat(m map[string]any, keys ...string) float64 {
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			return getFloatVal(cur[k])
		}
		next, ok := cur[k].(map[string]any)
		if !ok {
			return 0
		}
		cur = next
	}
	return 0
}

// getFloatVal converte any para float64 (aceita float64, int, int64).
func getFloatVal(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}
