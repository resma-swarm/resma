package tabs

import tea "github.com/charmbracelet/bubbletea"

// ServicesTab represents Tab [1] Services in the dashboard.
type ServicesTab struct{}

// NewServicesTab creates a new ServicesTab instance.
func NewServicesTab() *ServicesTab {
	return &ServicesTab{}
}

// Title returns the display title for this tab.
func (t ServicesTab) Title() string {
	return "Services"
}

// Init performs initial setup for the services tab.
func (t ServicesTab) Init() tea.Cmd {
	return nil
}

// Update handles messages for the services tab.
func (t ServicesTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the services tab as a string.
func (t ServicesTab) View() string {
	return ""
}
