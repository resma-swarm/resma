// Package mlclient implementa o HTTP client para o ML sidecar Python.
//
// Portado de backend/services/recommender.py (chamadas HTTP). O Go API
// chama o ML sidecar via HTTP com fallback graceful se indisponível.
package mlclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Client é o HTTP client para o ML sidecar.
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
	enabled    bool
}

// New cria um novo ML sidecar client.
func New(baseURL string, enabled bool) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log:     slog.Default().With("component", "ml-client"),
		enabled: enabled,
	}
}

// Health verifica se o ML sidecar está respondendo.
func (c *Client) Health(ctx context.Context) error {
	if !c.enabled {
		return fmt.Errorf("ML sidecar disabled")
	}
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ML sidecar health returned %d", resp.StatusCode)
	}
	return nil
}

// AnalyzeAll analisa todos os serviços. Retorna (result, error).
// Se o ML sidecar estiver indisponível, retorna error para o caller fazer fallback.
func (c *Client) AnalyzeAll(ctx context.Context) (any, error) {
	return c.get(ctx, "/analyze")
}

// AnalyzeService analisa um serviço específico.
func (c *Client) AnalyzeService(ctx context.Context, service string) (any, error) {
	return c.get(ctx, "/analyze/"+service)
}

// EvaluateTriggers avalia triggers de reanálise.
func (c *Client) EvaluateTriggers(ctx context.Context) (any, error) {
	return c.get(ctx, "/triggers")
}

// AnalyzeStorage analisa storage.
func (c *Client) AnalyzeStorage(ctx context.Context) (any, error) {
	return c.get(ctx, "/analyze/storage")
}

// Forecast forecast de memória para um serviço.
func (c *Client) Forecast(ctx context.Context, service string, days int) (any, error) {
	return c.get(ctx, fmt.Sprintf("/forecast/%s?days=%d", service, days))
}

// AlertsResult é o contrato retornado por GET /alerts no ML sidecar.
// Shape esperado pelo frontend Dashboard (leak_alerts / drift_alerts).
type AlertsResult struct {
	LeakAlerts  []map[string]any `json:"leak_alerts"`
	DriftAlerts []map[string]any `json:"drift_alerts"`
}

// GetAlerts busca memory leaks e resource drifts detectados pelo ML sidecar.
// Retorna nil e error se o sidecar estiver indisponível — o caller deve
// fazer fallback graceful retornando arrays vazios.
func (c *Client) GetAlerts(ctx context.Context) (*AlertsResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ML sidecar disabled")
	}
	url := c.baseURL + "/alerts"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Warn("ML sidecar indisponível (alerts)", "url", url, "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML sidecar returned %d: %s", resp.StatusCode, string(body))
	}
	var result AlertsResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ML alerts response: %w", err)
	}
	if result.LeakAlerts == nil {
		result.LeakAlerts = []map[string]any{}
	}
	if result.DriftAlerts == nil {
		result.DriftAlerts = []map[string]any{}
	}
	return &result, nil
}

// Enabled retorna se o ML sidecar está habilitado.
func (c *Client) Enabled() bool {
	return c.enabled
}

// get faz uma chamada GET ao ML sidecar.
func (c *Client) get(ctx context.Context, path string) (any, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ML sidecar disabled")
	}
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Warn("ML sidecar indisponível", "url", url, "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML sidecar returned %d: %s", resp.StatusCode, string(body))
	}
	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ML response: %w", err)
	}
	return result, nil
}
