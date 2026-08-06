package tabs

import tea "github.com/charmbracelet/bubbletea"

// AgentsTab represents Tab [3] Agents in the dashboard.
type AgentsTab struct{}

// NewAgentsTab creates a new AgentsTab instance.
func NewAgentsTab() *AgentsTab {
	return &AgentsTab{}
}

// Title returns the display title for this tab.
func (t AgentsTab) Title() string {
	return "Agents"
}

// Init performs initial setup for the agents tab.
func (t AgentsTab) Init() tea.Cmd {
	return nil
}

// Update handles messages for the agents tab.
func (t AgentsTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the agents tab as a string.
func (t AgentsTab) View() string {
	return ""
}
