// Package tui contains the RESMA interactive TUI monitor dashboard.
// It provides a Bubble Tea-based terminal UI with 6 tabs (Services, Nodes,
// Agents, Tasks, Alerts, Recommendations), drill-down detail views, logs,
// column sorting, and a k9s-inspired layout.
//
// Run is the entry point called by the `resma monitor` CLI command.
package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the interactive TUI monitor in the terminal (alt screen).
// Returns an error if the Bubble Tea program fails to initialize or run.
func Run() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}
