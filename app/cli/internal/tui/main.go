// Package tui contains the RESMA interactive TUI monitor dashboard.
// It provides a Bubble Tea-based terminal UI with 6 tabs (Services, Nodes,
// Agents, Tasks, Alerts, Recommendations), drill-down detail views, logs,
// column sorting, and a RESMA-style layout.
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
// It loads persisted credentials, fetches the initial dashboard data via
// REST, then connects to the SSE stream for live updates.
// Returns an error if the Bubble Tea program fails to initialize or run.
func Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api, err := client.LoadOrRefresh(ctx)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	m := initialModel()

	// Criar o programa Bubble Tea
	p := tea.NewProgram(m, tea.WithAltScreen())

	// 1. Carregar dados iniciais via REST (não esperar até 60s pelo SSE)
	go fetchInitialDashboard(ctx, p, api)

	// 2. Iniciar goroutine SSE para updates em tempo real
	go startSSE(ctx, p, api)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

// fetchInitialDashboard faz um GET /api/dashboard para obter dados imediatamente.
// O SSE só publica a cada 60s (ClusterInterval), então sem isso o usuário
// ficaria até 60s sem dados ao entrar no monitor.
func fetchInitialDashboard(ctx context.Context, p *tea.Program, api *client.APIClient) {
	var payload client.DashboardPayload
	if err := api.Get(ctx, "/api/dashboard", &payload); err != nil {
		// Se falhar, o SSE vai tentar suprir os dados depois
		p.Send(sseErrorMsg{err: fmt.Errorf("initial fetch: %w", err)})
		return
	}
	p.Send(clusterHealthMsg{payload: payload})
}

// startSSE conecta ao tópico "dashboard" do SSE e envia clusterHealthMsg
// para o programa Bubble Tea a cada evento recebido.
// Roda em goroutine — termina quando ctx é cancelado.
func startSSE(ctx context.Context, p *tea.Program, api *client.APIClient) {
	sse := client.NewSSEClient(api.BaseURL, api.Token(), "dashboard", func(event client.SSEEvent) error {
		// Só processar eventos "cluster" (o tópico dashboard também recebe
		// eventos "storage" com payload diferente — ignorar para não zerar
		// os campos de cluster_capacity)
		if event.Type != "cluster" {
			return nil
		}

		// Parsear payload do dashboard
		var payload client.DashboardPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil // payload inválido — ignorar
		}

		// Enviar msg para o Bubble Tea
		p.Send(clusterHealthMsg{payload: payload})
		return nil
	})

	if err := sse.Run(ctx); err != nil && ctx.Err() == nil {
		p.Send(sseErrorMsg{err: err})
	}
}
