// Package client — APIClient base com auth JWT e auto-refresh.
//
// APIClient é usado por todos os comandos do CLI (services, nodes, monitor, etc).
// Ele encapsula baseURL, token e http.Client. Se um request retorna 401,
// tenta automaticamente renovar o access token via refresh token.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient é o cliente HTTP para o Go API do RESMA.
type APIClient struct {
	BaseURL string
	creds   *Credentials
	http    *http.Client
}

// New cria um APIClient com as credenciais persistidas.
// Se creds for nil, o cliente opera sem auth (apenas endpoints públicos).
func New(creds *Credentials) *APIClient {
	baseURL := "http://localhost:8080"
	if creds != nil && creds.ServerURL != "" {
		baseURL = creds.ServerURL
	}
	return &APIClient{
		BaseURL: baseURL,
		creds:   creds,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewWithURL cria um APIClient apontando para um servidor específico (login).
func NewWithURL(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Token retorna o access token atual (para SSE clients).
func (c *APIClient) Token() string {
	if c.creds == nil {
		return ""
	}
	return c.creds.AccessToken
}

// Credentials retorna as credenciais atuais.
func (c *APIClient) Credentials() *Credentials {
	return c.creds
}

// SetCredentials atualiza as credenciais (após login ou refresh).
func (c *APIClient) SetCredentials(creds *Credentials) {
	c.creds = creds
}

// APIError representa um erro retornado pelo API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
}

// IsUnauthorized retorna true se o erro é 401 (token expirado).
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// do executa um request HTTP com auth JWT. Se receber 401 e houver refresh
// token, tenta renovar e retentar uma vez.
func (c *APIClient) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	resp, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	// Se 401 e temos refresh token, tentar renovar e retentar
	if resp.StatusCode == http.StatusUnauthorized && c.creds != nil && c.creds.IsRefreshable() {
		resp.Body.Close()
		if err := c.refreshToken(ctx); err != nil {
			return nil, fmt.Errorf("token expired, refresh failed: %w", err)
		}
		return c.doRaw(ctx, method, path, body)
	}

	return resp, nil
}

// doRaw executa um request HTTP sem retry.
func (c *APIClient) doRaw(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.creds != nil && c.creds.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.creds.AccessToken)
	}

	return c.http.Do(req)
}

// Get faz um GET autenticado e decodifica a resposta JSON.
func (c *APIClient) Get(ctx context.Context, path string, out any) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

// Post faz um POST autenticado e decodifica a resposta JSON.
func (c *APIClient) Post(ctx context.Context, path string, body, out any) error {
	resp, err := c.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

// PostUnauth faz um POST sem auth (usado por login/refresh).
func (c *APIClient) PostUnauth(ctx context.Context, path string, body, out any) error {
	resp, err := c.doRaw(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

// decodeResponse lê o body e decodifica JSON. Se status >= 400, retorna APIError.
func decodeResponse(resp *http.Response, out any) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// refreshToken renova o access token usando o refresh token.
// Atualiza creds em memória e persiste no disco.
func (c *APIClient) refreshToken(ctx context.Context) error {
	if c.creds == nil || !c.creds.IsRefreshable() {
		return fmt.Errorf("no refresh token available")
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		TokenType    string `json:"token_type"`
	}
	// /api/auth/refresh aceita refresh_token no body
	body := map[string]string{"refresh_token": c.creds.RefreshToken}
	if err := c.PostUnauth(ctx, "/api/auth/refresh", body, &result); err != nil {
		return err
	}

	c.creds.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		c.creds.RefreshToken = result.RefreshToken
	}
	c.creds.TokenType = result.TokenType
	// Access token TTL default é 15min — conservador
	c.creds.ExpiresAt = time.Now().Add(15 * time.Minute)

	return SaveCredentials(c.creds)
}
