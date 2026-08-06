// Package tui contains the RESMA interactive TUI monitor dashboard.
// It provides a Bubble Tea-based terminal UI with 6 tabs (Services, Nodes,
// Agents, Tasks, Alerts, Recommendations), drill-down detail views, logs,
// column sorting, and a k9s-inspired layout.
//
// Run is the entry point called by the `resma monitor` CLI command.
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/resma-swarm/resma/app/cli/internal/client"
)

// Run starts the interactive TUI monitor in the terminal (alt screen).
// It loads persisted credentials, connects to the SSE dashboard stream,
// and renders the dashboard with real cluster data.
// Returns an error if the Bubble Tea program fails to initialize or run.
func Run() error {
	// Carregar credenciais persistidas
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api, err := client.LoadOrRefresh(ctx)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	m := initialModel()

	// Criar o programa Bubble Tea (precisamos do p para enviar msgs da goroutine)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Iniciar goroutine SSE — envia clusterHealthMsg para o programa
	go startSSE(ctx, p, api)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

// startSSE conecta ao tópico "dashboard" do SSE e envia clusterHealthMsg
// para o programa Bubble Tea a cada evento recebido.
// Roda em goroutine — termina quando ctx é cancelado.
func startSSE(ctx context.Context, p *tea.Program, api *client.APIClient) {
	sse := client.NewSSEClient(api.BaseURL, api.Token(), "dashboard", func(event client.SSEEvent) error {
		// Evento "connected" é o handshake inicial — ignorar
		if event.Type == "connected" {
			return nil
		}

		// Parsear payload do dashboard
		var payload client.DashboardPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			// Payload inválido — ignorar silenciosamente
			return nil
		}

		// Enviar msg para o Bubble Tea
		p.Send(clusterHealthMsg{payload: payload})
		return nil
	})

	// Run bloqueia até ctx ser cancelado. Se a conexão cair, reconecta
	// automaticamente com backoff exponencial.
	if err := sse.Run(ctx); err != nil && ctx.Err() == nil {
		// Erro real (não cancelamento) — notificar o TUI
		p.Send(sseErrorMsg{err: err})
	}
}
