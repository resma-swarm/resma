package tabs

import tea "github.com/charmbracelet/bubbletea"

// NodesTab represents Tab [2] Nodes in the dashboard.
type NodesTab struct{}

// NewNodesTab creates a new NodesTab instance.
func NewNodesTab() *NodesTab {
	return &NodesTab{}
}

// Title returns the display title for this tab.
func (t NodesTab) Title() string {
	return "Nodes"
}

// Init performs initial setup for the nodes tab.
func (t NodesTab) Init() tea.Cmd {
	return nil
}

// Update handles messages for the nodes tab.
func (t NodesTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the nodes tab as a string.
func (t NodesTab) View() string {
	return ""
}
