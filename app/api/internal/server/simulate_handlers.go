// Package server — /api/recommendations/simulate + /api/recommendations/export-yaml
// (Right-Sizing Studio R6 — simulação em lote + export YAML declarativo).
package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// registerSimulateRoutes registra as rotas de simulate + export-yaml.
func (s *Server) registerSimulateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/recommendations/simulate", s.handleSimulateBatch)
	mux.HandleFunc("GET /api/recommendations/export-yaml", s.handleExportYAML)
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
		"tier":          tier,
		"simulated_at":  time.Now().UTC().Format(time.RFC3339),
		"rows":          rows,
		"totals": map[string]any{
			"services":        validCount,
			"cpu_cores_freed": totalCPUCores,
			"mem_bytes_freed": totalMemBytes,
		},
	})
}

// handleExportYAML gera um plano YAML declarativo (GitOps-safe) das recomendações.
//
// Spec: export-yaml.md §1 GET /api/recommendations/export-yaml
func (s *Server) handleExportYAML(w http.ResponseWriter, r *http.Request) {
	tier := r.URL.Query().Get("tier")
	if tier == "" {
		tier = "balanced"
	}
	if tier != "conservative" && tier != "balanced" && tier != "aggressive" {
		writeError(w, http.StatusBadRequest, "tier must be one of: conservative, balanced, aggressive")
		return
	}
	servicesFilter := r.URL.Query().Get("services")
	var filterSet map[string]bool
	if servicesFilter != "" {
		filterSet = make(map[string]bool)
		for _, s := range strings.Split(servicesFilter, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				filterSet[s] = true
			}
		}
	}

	ctx := r.Context()
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

	type svcEntry struct {
		Name      string                 `yaml:"name"`
		Current   map[string]string      `yaml:"current"`
		Suggested map[string]string      `yaml:"suggested"`
		Freed     map[string]any         `yaml:"freed"`
		Risk      map[string]any         `yaml:"risk"`
		Pattern   string                 `yaml:"pattern"`
		Confidence string                `yaml:"confidence"`
	}

	type planYAML struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		Metadata   struct {
			GeneratedAt   string         `yaml:"generatedAt"`
			Tier          string         `yaml:"tier"`
			ServiceCount  int            `yaml:"serviceCount"`
			TotalFreed    map[string]any `yaml:"totalFreed"`
		} `yaml:"metadata"`
		Spec struct {
			Services []svcEntry `yaml:"services"`
		} `yaml:"spec"`
	}

	plan := planYAML{
		APIVersion: "resma/v1",
		Kind:       "RightSizingPlan",
	}
	plan.Metadata.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	plan.Metadata.Tier = tier
	plan.Metadata.TotalFreed = map[string]any{"cpuCores": 0.0, "memBytes": int64(0)}

	var totalCPU float64
	var totalMem int64

	for _, r := range recs {
		rec, ok := r.(map[string]any)
		if !ok {
			continue
		}
		svcName, _ := rec["service"].(string)
		if svcName == "" {
			continue
		}
		// Filtrar por services param se fornecido
		if filterSet != nil && !filterSet[svcName] {
			continue
		}

		// Buscar current do Docker
		current, _ := s.docker.GetServiceResources(ctx, svcName)

		// Buscar suggested_tiers[tier]
		tiers, _ := rec["suggested_tiers"].(map[string]any)
		tierData, _ := tiers[tier].(map[string]any)
		if tierData == nil {
			continue // sem suggested_tiers — pular
		}

		suggestedCPU := getFloatVal(tierData["cpu_limit"])
		suggestedMem := getFloatVal(tierData["mem_limit"])
		suggestedCPURes := getFloatVal(tierData["cpu_reservation"])
		suggestedMemRes := getFloatVal(tierData["mem_reservation"])

		// freed
		var freedCPU, freedMem float64
		var freedCPUPct, freedMemPct int
		if current.CPULimit > 0 && current.CPULimit > suggestedCPU {
			freedCPU = current.CPULimit - suggestedCPU
			freedCPUPct = int((freedCPU / current.CPULimit) * 100)
		}
		if current.MemLimit > 0 && float64(current.MemLimit) > suggestedMem {
			freedMem = float64(current.MemLimit) - suggestedMem
			freedMemPct = int((freedMem / float64(current.MemLimit)) * 100)
		}

		totalCPU += freedCPU
		totalMem += int64(freedMem)

		// Risk
		riskMap := map[string]any{"level": "unknown", "score": 0}
		if r, ok := rec["risk"].(map[string]any); ok {
			riskMap = r
		}

		pattern, _ := rec["pattern_label"].(string)
		if pattern == "" {
			pattern = "unknown"
		}
		confidence, _ := rec["confidence"].(string)
		if confidence == "" {
			confidence = "medium"
		}

		entry := svcEntry{
			Name: svcName,
			Current: map[string]string{
				"cpuLimit":       fmt.Sprintf("%.2f", current.CPULimit),
				"memLimit":       fmt.Sprintf("%d", current.MemLimit),
				"cpuReservation": fmt.Sprintf("%.2f", current.CPUReservation),
				"memReservation": fmt.Sprintf("%d", current.MemReservation),
			},
			Suggested: map[string]string{
				"cpuLimit":       fmt.Sprintf("%.2f", suggestedCPU),
				"memLimit":       fmt.Sprintf("%d", int64(suggestedMem)),
				"cpuReservation": fmt.Sprintf("%.2f", suggestedCPURes),
				"memReservation": fmt.Sprintf("%d", int64(suggestedMemRes)),
			},
			Freed: map[string]any{
				"cpuCores": freedCPU,
				"memBytes": int64(freedMem),
				"cpuPct":   freedCPUPct,
				"memPct":   freedMemPct,
			},
			Risk:       riskMap,
			Pattern:    pattern,
			Confidence: confidence,
		}
		plan.Spec.Services = append(plan.Spec.Services, entry)
	}

	if len(plan.Spec.Services) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	plan.Metadata.ServiceCount = len(plan.Spec.Services)
	plan.Metadata.TotalFreed["cpuCores"] = totalCPU
	plan.Metadata.TotalFreed["memBytes"] = totalMem

	out, err := yaml.Marshal(plan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("yaml marshal: %v", err))
		return
	}

	// Header de comentários
	header := fmt.Sprintf("# RESMA Right-Sizing Plan — generated %s\n# Tier: %s\n# Services: %d\n# Total resources freed: %.2f cores, %s RAM\n\n",
		plan.Metadata.GeneratedAt, tier, plan.Metadata.ServiceCount,
		totalCPU, formatBytesYAML(totalMem))

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", `attachment; filename="resma-right-sizing-plan.yaml"`)
	w.Write([]byte(header))
	w.Write(out)
}

// --- helpers ---

// getFloat navega em map aninhado e retorna um float.
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

// formatBytesYAML formata bytes como string legível para o header do YAML.
func formatBytesYAML(n int64) string {
	if n >= 1e9 {
		return fmt.Sprintf("%.1fGB", float64(n)/1e9)
	}
	if n >= 1e6 {
		return fmt.Sprintf("%.0fMB", float64(n)/1e6)
	}
	return fmt.Sprintf("%dB", n)
}
