package components

import "github.com/charmbracelet/lipgloss"

// TabBar renders the horizontal tab navigation bar.
type TabBar struct {
	tabs   []string
	active int
	width  int
	style  lipgloss.Style
}

// NewTabBar creates a new TabBar component.
func NewTabBar() *TabBar {
	return &TabBar{}
}

// View renders the tab bar as a string.
func (t TabBar) View() string {
	return ""
}
