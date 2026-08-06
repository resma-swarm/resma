package components

import "github.com/charmbracelet/lipgloss"

// Header renders the top banner of the dashboard with title and status.
type Header struct {
	title  string
	status string
	width  int
	style  lipgloss.Style
}

// NewHeader creates a new Header component.
func NewHeader() *Header {
	return &Header{}
}

// View renders the header as a string.
func (h Header) View() string {
	return ""
}
