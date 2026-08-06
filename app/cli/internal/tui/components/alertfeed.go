package components

import "github.com/charmbracelet/lipgloss"

// AlertFeed renders a scrolling feed of alerts sorted by severity.
type AlertFeed struct {
	alerts []AlertItem
	width  int
	style  lipgloss.Style
}

// AlertItem represents a single alert in the feed.
type AlertItem struct {
	ID       string
	Severity string
	Message  string
	Source   string
}

// NewAlertFeed creates a new AlertFeed component.
func NewAlertFeed() *AlertFeed {
	return &AlertFeed{}
}

// View renders the alert feed as a string.
func (a AlertFeed) View() string {
	return ""
}
