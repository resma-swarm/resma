package components

import "github.com/charmbracelet/lipgloss"

// ServiceTable renders a table of services with status and health columns.
type ServiceTable struct {
	rows  []ServiceRow
	width int
	style lipgloss.Style
}

// ServiceRow represents a single row in the service table.
type ServiceRow struct {
	Name    string
	Status  string
	Health  string
	Latency string
}

// NewServiceTable creates a new ServiceTable component.
func NewServiceTable() *ServiceTable {
	return &ServiceTable{}
}

// View renders the service table as a string.
func (s ServiceTable) View() string {
	return ""
}
