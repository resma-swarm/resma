package tabs

import tea "github.com/charmbracelet/bubbletea"

// AlertsTab represents Tab [5] Alerts in the dashboard.
type AlertsTab struct{}

// NewAlertsTab creates a new AlertsTab instance.
func NewAlertsTab() *AlertsTab {
	return &AlertsTab{}
}

// Title returns the display title for this tab.
func (t AlertsTab) Title() string {
	return "Alerts"
}

// Init performs initial setup for the alerts tab.
func (t AlertsTab) Init() tea.Cmd {
	return nil
}

// Update handles messages for the alerts tab.
func (t AlertsTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the alerts tab as a string.
func (t AlertsTab) View() string {
	return ""
}
