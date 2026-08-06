package components

import "github.com/charmbracelet/lipgloss"

// Sparkline renders a compact inline chart of numeric values.
type Sparkline struct {
	values []float64
	width  int
	style  lipgloss.Style
}

// NewSparkline creates a new Sparkline component.
func NewSparkline() *Sparkline {
	return &Sparkline{}
}

// View renders the sparkline as a string.
func (s Sparkline) View() string {
	return ""
}
