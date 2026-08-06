// Package client — SSE client reutilizável para streams do RESMA API.
//
// SSEClient abre uma conexão HTTP persistente com /api/sse/{topic}, lê o
// stream linha por linha, parseia eventos no formato "event: type\ndata: json\n\n"
// e chama um callback para cada evento. É reutilizável por qualquer comando
// do CLI que precise de streaming (monitor, resma services --watch, etc).
//
// O client suporta reconexão automática com backoff exponencial se a conexão
// cair. O contexto controla o ciclo de vida — cancelar o ctx fecha o stream.
package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SSEEvent representa um evento SSE parseado do stream.
type SSEEvent struct {
	Type string // valor do campo "event:" (ex: "dashboard", "connected")
	Data []byte // valor do campo "data:" (payload JSON bruto)
}

// SSEHandler é o callback chamado para cada evento recebido.
// Recebe o evento parseado. Se retornar erro, o stream para.
type SSEHandler func(SSEEvent) error

// SSEClient gerencia uma conexão SSE com o RESMA API.
type SSEClient struct {
	BaseURL string // URL base do API (ex: http://localhost:8080)
	Token   string // JWT access token (Bearer)
	Topic   string // tópico SSE (ex: "dashboard", "metrics", "services")

	onEvent SSEHandler
	http    *http.Client
}

// NewSSEClient cria um SSE client para um tópico.
// O callback onEvent é chamado para cada evento recebido.
func NewSSEClient(baseURL, token, topic string, onEvent SSEHandler) *SSEClient {
	return &SSEClient{
		BaseURL: baseURL,
		Token:   token,
		Topic:   topic,
		onEvent: onEvent,
		http: &http.Client{
			Timeout: 0, // sem timeout — stream persistente
		},
	}
}

// Run abre a conexão SSE e processa eventos até o contexto ser cancelado.
// Se a conexão cair, tenta reconectar com backoff exponencial (1s, 2s, 4s, ...,
// max 30s). Retorna apenas quando ctx.Done().
func (c *SSEClient) Run(ctx context.Context) error {
	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := c.connectAndServe(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			// Conexão caiu ou erro de parse — reconectar
			_ = err // logar se necessário
		}

		// Backoff antes de reconectar
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// connectAndServe abre uma conexão SSE e processa eventos até erro/desconexão.
func (c *SSEClient) connectAndServe(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/sse/%s", c.BaseURL, c.Topic)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE connect: HTTP %d", resp.StatusCode)
	}

	return c.parseStream(ctx, resp.Body)
}

// parseStream lê o stream SSE linha por linha e monta eventos.
// Formato SSE:
//
//	event: dashboard
//	data: {"cluster_capacity": {...}}
//
// (linha em branco separa eventos)
// Linhas começando com ":" são comentários (keepalive ping).
func (c *SSEClient) parseStream(ctx context.Context, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	// Buffer generoso — payloads de dashboard podem ser grandes
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var dataLines []string

	flush := func() error {
		if len(dataLines) == 0 {
			eventType = ""
			return nil
		}
		event := SSEEvent{
			Type: eventType,
			Data: []byte(strings.Join(dataLines, "\n")),
		}
		if err := c.onEvent(event); err != nil {
			return err
		}
		eventType = ""
		dataLines = dataLines[:0]
		return nil
	}

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line := scanner.Text()

		// Linha vazia = fim do evento
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}

		// Comentário (keepalive ping ": ping")
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Campo: valor
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue // linha malformada
		}
		field := line[:colon]
		value := line[colon+1:]
		// Space após ':' é opcional — remover 1 espaço se presente
		if strings.HasPrefix(value, " ") {
			value = value[1:]
		}

		switch field {
		case "event":
			eventType = value
		case "data":
			dataLines = append(dataLines, value)
		// "id" e "retry" não usados pelo RESMA
		}
	}

	// Flush final (se houver evento pendente)
	if err := flush(); err != nil {
		return err
	}
	return scanner.Err()
}
