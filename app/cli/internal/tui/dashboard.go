package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/resma-swarm/resma/app/cli/internal/tui/tabs"
)

// DashboardModel is the main Bubble Tea model for the monitor command.
// It manages the tab bar and delegates rendering to the active tab model.
type DashboardModel struct {
	activeTab int
	width     int
	height    int
	tabs      []tabs.Tab
	styles    Styles
}

// Model returns a new DashboardModel instance.
func Model() tea.Model {
	return DashboardModel{}
}

// Init performs initial setup for the dashboard model.
func (m DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the dashboard model accordingly.
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View renders the dashboard model as a string.
func (m DashboardModel) View() string {
	return ""
}
