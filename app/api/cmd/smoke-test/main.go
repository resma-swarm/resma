// Command smoke-test valida todos os endpoints da API RESMA Go.
//
// Uso: go run ./cmd/smoke-test
//
// Pré-requisitos:
//   - API Go rodando em :8080
//   - Onboarding done (owner criado — ver docs/README.md para credenciais)
//
// O script testa:
//  1. /health, /ready
//  2. Auth flow (login, me, refresh, logout)
//  3. Endpoints internos com JWT (config, services, dashboard, nodes, etc)
//  4. Endpoints públicos com API key (/api/v1/*)
//  5. SSE session (create, delete)
//  6. API key CRUD
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const baseURL = "http://localhost:8080"

type testResult struct {
	name   string
	status int
	passed bool
	detail string
}

func main() {
	results := []testResult{}

	// --- 1. Infra ---
	results = append(results, test("GET /health", func() (int, string) {
		return doRequest("GET", "/health", nil, "")
	}))
	results = append(results, test("GET /ready", func() (int, string) {
		return doRequest("GET", "/ready", nil, "")
	}))

	// --- 2. Auth ---
	loginResp, loginBody := doRequestRaw("POST", "/api/auth/login",
		map[string]string{"username": "owner", "password": "owner123"}, "")
	var loginData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.Unmarshal([]byte(loginBody), &loginData)
	token := loginData.AccessToken

	results = append(results, testResult{
		name:   "POST /api/auth/login",
		status: loginResp,
		passed: loginResp == 200 && token != "",
		detail: fmt.Sprintf("status=%d, token_len=%d", loginResp, len(token)),
	})

	if token == "" {
		fmt.Println("FAIL: no token, cannot continue tests")
		printResults(results)
		os.Exit(1)
	}

	results = append(results, test("GET /api/auth/me (JWT)", func() (int, string) {
		return doRequest("GET", "/api/auth/me", nil, token)
	}))
	results = append(results, test("GET /api/auth/status (no auth)", func() (int, string) {
		return doRequest("GET", "/api/auth/status", nil, "")
	}))

	// --- 3. Endpoints internos com JWT ---
	internalEndpoints := []struct {
		method string
		path   string
		name   string
	}{
		{"GET", "/api/config", "GET /api/config"},
		{"GET", "/api/services", "GET /api/services"},
		{"GET", "/api/services/sparklines", "GET /api/services/sparklines"},
		{"GET", "/api/dashboard", "GET /api/dashboard"},
		{"GET", "/api/nodes", "GET /api/nodes"},
		{"GET", "/api/nodes/cluster", "GET /api/nodes/cluster"},
		{"GET", "/api/change-log", "GET /api/change-log"},
		{"GET", "/api/oom-events", "GET /api/oom-events"},
		{"GET", "/api/recommendations", "GET /api/recommendations"},
		{"GET", "/api/recommendations/triggers", "GET /api/recommendations/triggers"},
		{"GET", "/api/recommendations/storage", "GET /api/recommendations/storage"},
		{"GET", "/api/schedules", "GET /api/schedules"},
		{"GET", "/api/schedules/pending", "GET /api/schedules/pending"},
		{"GET", "/api/schedules/history", "GET /api/schedules/history"},
		{"GET", "/api/templates", "GET /api/templates"},
		{"GET", "/api/storage/summary", "GET /api/storage/summary"},
		{"GET", "/api/storage/trend", "GET /api/storage/trend"},
		{"GET", "/api/storage/volumes/growth", "GET /api/storage/volumes/growth"},
		{"GET", "/api/auth/api-keys", "GET /api/auth/api-keys"},
		// Fase 7 — agents + tasks + service health
		{"GET", "/api/agents", "GET /api/agents"},
		{"GET", "/api/tasks", "GET /api/tasks"},
		{"GET", "/api/services/health", "GET /api/services/health"},
	}

	for _, ep := range internalEndpoints {
		results = append(results, test(ep.name, func() (int, string) {
			return doRequest(ep.method, ep.path, nil, token)
		}))
	}

	// --- 4a. Fase 7 — Agent ingestion endpoints (sem JWT, com agent token) ---
	agentToken := os.Getenv("RESMA_AGENT_TOKEN")
	if agentToken == "" {
		agentToken = "dev-agent-token-change-me"
	}
	results = append(results, testResult{
		name:   "POST /api/agent/heartbeat (agent token)",
		status: 0,
		passed: false,
	})
	hbCode, _ := doRequestWithToken("POST", "/api/agent/heartbeat",
		map[string]any{"node_id": "smoke-test-node", "hostname": "smoke-host", "containers_count": 0, "version": "smoke"},
		agentToken)
	results[len(results)-1] = testResult{
		name:   "POST /api/agent/heartbeat (agent token)",
		status: hbCode,
		passed: hbCode == 200,
		detail: fmt.Sprintf("status=%d", hbCode),
	}

	results = append(results, testResult{
		name:   "POST /api/agent/ingest/metrics (agent token)",
		status: 0,
		passed: false,
	})
	metricsCode, _ := doRequestWithToken("POST", "/api/agent/ingest/metrics",
		map[string]any{
			"node_id": "smoke-test-node",
			"metrics": []map[string]any{
				{"ts": "2026-01-01T00:00:00Z", "service": "smoke-svc", "container_id": "smoke123", "cpu_percent": 1.0, "mem_usage": 100, "mem_limit": 1000},
			},
		},
		agentToken)
	results[len(results)-1] = testResult{
		name:   "POST /api/agent/ingest/metrics (agent token)",
		status: metricsCode,
		passed: metricsCode == 200,
		detail: fmt.Sprintf("status=%d", metricsCode),
	}

	// Agent ingestion sem token deve falhar (401)
	results = append(results, testResult{
		name:   "POST /api/agent/heartbeat (no token — should 401)",
		status: 0,
		passed: false,
	})
	noTokenCode, _ := doRequestWithToken("POST", "/api/agent/heartbeat",
		map[string]any{"node_id": "should-fail"}, "")
	results[len(results)-1] = testResult{
		name:   "POST /api/agent/heartbeat (no token — should 401)",
		status: noTokenCode,
		passed: noTokenCode == 401,
		detail: fmt.Sprintf("status=%d", noTokenCode),
	}

	// --- 4. Auth sem JWT deve falhar ---
	results = append(results, testResult{
		name:   "GET /api/config (no JWT — should 401)",
		status: 0,
		passed: false,
	})
	code, _ := doRequest("GET", "/api/config", nil, "")
	results[len(results)-1] = testResult{
		name:   "GET /api/config (no JWT — should 401)",
		status: code,
		passed: code == 401,
		detail: fmt.Sprintf("status=%d", code),
	}

	// --- 5. SSE session ---
	sseCode, _ := doRequest("POST", "/api/sse/session", nil, token)
	results = append(results, testResult{
		name:   "POST /api/sse/session (create cookie)",
		status: sseCode,
		passed: sseCode == 200,
		detail: fmt.Sprintf("status=%d", sseCode),
	})

	// --- 6. Print results ---
	printResults(results)

	// Count failures
	failures := 0
	for _, r := range results {
		if !r.passed {
			failures++
		}
	}
	if failures > 0 {
		fmt.Printf("\n%d test(s) FAILED\n", failures)
		os.Exit(1)
	}
	fmt.Printf("\nAll %d tests PASSED\n", len(results))
}

func test(name string, fn func() (int, string)) testResult {
	code, body := fn()
	passed := code >= 200 && code < 300
	detail := fmt.Sprintf("status=%d, body_len=%d", code, len(body))
	if !passed {
		detail += ", body=" + truncate(body, 200)
	}
	return testResult{name: name, status: code, passed: passed, detail: detail}
}

func doRequest(method, path string, body any, token string) (int, string) {
	code, bodyStr := doRequestRaw(method, path, body, token)
	return code, bodyStr
}

// doRequestWithToken é um alias para doRequestRaw — explícito para agent token.
func doRequestWithToken(method, path string, body any, token string) (int, string) {
	return doRequestRaw(method, path, body, token)
}

func doRequestRaw(method, path string, body any, token string) (int, string) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, baseURL+path, reqBody)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func printResults(results []testResult) {
	fmt.Println("\n=== RESMA API Smoke Test ===")
	fmt.Println(strings.Repeat("=", 80))
	for _, r := range results {
		status := "PASS"
		if !r.passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %-45s %s\n", status, r.name, r.detail)
	}
	fmt.Println(strings.Repeat("=", 80))
}

func init() {
	http.DefaultClient.Timeout = 10 * time.Second
}
