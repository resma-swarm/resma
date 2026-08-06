// Command wire-tui runs a high-fidelity wireframe of the RESMA CLI dashboard
// with mock data. It demonstrates the production layout, two-column panels,
// 6 tabs, drill-down, filter/command modes, and visual style.
//
// Usage: go run ./cmd/wire-tui
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
