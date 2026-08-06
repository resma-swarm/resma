package components

import "github.com/charmbracelet/lipgloss"

// Footer renders the bottom status bar with keybindings and status info.
type Footer struct {
	width int
	style lipgloss.Style
}

// NewFooter creates a new Footer component.
func NewFooter() *Footer {
	return &Footer{}
}

// View renders the footer as a string.
func (f Footer) View() string {
	return ""
}
