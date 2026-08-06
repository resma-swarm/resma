package components

import "github.com/charmbracelet/lipgloss"

// MetricCard displays a single labeled metric value with optional trend.
type MetricCard struct {
	label string
	value string
	trend string
	style lipgloss.Style
}

// NewMetricCard creates a new MetricCard component.
func NewMetricCard() *MetricCard {
	return &MetricCard{}
}

// View renders the metric card as a string.
func (m MetricCard) View() string {
	return ""
}
