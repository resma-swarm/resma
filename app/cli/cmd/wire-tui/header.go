package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderHeader(m model) string {
	title := sHeader.Render("RESMA")
	subtitle := sSubtitle.Render("  Docker Swarm Resource Manager")
	clock := sMuted.Render(m.clock.Format("2006-01-02 15:04:05"))

	status := sSuccess.Render("● Online")
	if m.viewMode == ViewFilter || m.viewMode == ViewCommand {
		status = sWarning.Render("● " + modeName(m.viewMode))
	}

	padding := strings.Repeat(" ", max(0, m.width-65))
	return lipgloss.JoinHorizontal(lipgloss.Top, title, subtitle, padding, clock, "  ", status)
}

func modeName(mode ViewMode) string {
	switch mode {
	case ViewFilter:
		return "Filter"
	case ViewCommand:
		return "Command"
	case ViewDetail:
		return "Detail"
	case ViewHelp:
		return "Help"
	default:
		return "Online"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
