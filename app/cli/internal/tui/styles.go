package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the Lipgloss style definitions used by the dashboard.
type Styles struct {
	HeaderStyle      lipgloss.Style
	TabBarStyle      lipgloss.Style
	TabActiveStyle   lipgloss.Style
	TabInactiveStyle lipgloss.Style
	FooterStyle      lipgloss.Style
	ContentStyle     lipgloss.Style
	TitleStyle       lipgloss.Style
	SubtitleStyle    lipgloss.Style
	BorderStyle      lipgloss.Style
	HighlightStyle   lipgloss.Style
	ErrorStyle       lipgloss.Style
	SuccessStyle     lipgloss.Style
	WarningStyle     lipgloss.Style
	MutedStyle       lipgloss.Style
}

// NewStyles returns a Styles struct initialized with default styles.
func NewStyles() Styles {
	return Styles{
		HeaderStyle: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),
		TabBarStyle: lipgloss.NewStyle().
			Padding(0, 1),
		TabActiveStyle: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 2),
		TabInactiveStyle: lipgloss.NewStyle().
			Padding(0, 2),
		FooterStyle: lipgloss.NewStyle().
			Padding(0, 1),
		ContentStyle:  lipgloss.NewStyle(),
		TitleStyle:    lipgloss.NewStyle().Bold(true),
		SubtitleStyle: lipgloss.NewStyle().Faint(true),
		BorderStyle:   lipgloss.NewStyle().BorderLeft(true),
		HighlightStyle: lipgloss.NewStyle().
			Bold(true),
		ErrorStyle: lipgloss.NewStyle().
			Bold(true),
		SuccessStyle: lipgloss.NewStyle().
			Bold(true),
		WarningStyle: lipgloss.NewStyle().
			Bold(true),
		MutedStyle: lipgloss.NewStyle().Faint(true),
	}
}
