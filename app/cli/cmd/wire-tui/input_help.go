package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderCommandInput(m model) string {
	prompt := sHighlight.Render(":")
	input := m.inputBuf + "_"
	hint := sMuted.Render("  (services, nodes, agents, tasks, alerts, recs, q, help)")
	return "\n\n" + prompt + " " + input + hint + "\n\n" +
		sMuted.Render(strings.Repeat("─", m.width-4))
}

func renderFilterInput(m model) string {
	prompt := sHighlight.Render("/")
	input := m.inputBuf + "_"
	hint := sMuted.Render("  (regex — Enter to apply, Esc to cancel)")
	return "\n\n" + prompt + " " + input + hint + "\n\n" +
		sMuted.Render(strings.Repeat("─", m.width-4))
}

func renderHelp(m model) string {
	var sb strings.Builder
	sb.WriteString(sK9sClusterTitle.Render("  KEYBINDINGS — " + tabNames[m.activeTab][4:] + " Tab"))
	sb.WriteString("\n\n")

	sections := [][]string{
		{"Navigation:", ""},
		{"j/k or up/down", "Move cursor"},
		{"g/G", "Go to top/bottom"},
		{"Enter", "Drill-down (detail view)"},
		{"Esc", "Back to list view"},
		{"Tab/Shift+Tab", "Switch panel (side <-> main)"},
		{"", ""},
		{"Actions:", ""},
		{"a", "Apply recommendation"},
		{"d", "Delete/drain (with confirmation)"},
		{"e", "Edit config"},
		{"l", "View logs"},
		{"s", "Shell into container"},
		{"y", "YAML/describe"},
		{"", ""},
		{"Filter:", ""},
		{"/", "Enter filter mode (regex)"},
		{"n/N", "Next/previous filter match"},
		{"", ""},
		{"Global:", ""},
		{"q", "Quit"},
		{"1-6", "Switch tab"},
		{":", "Command mode"},
		{"?", "This help"},
		{"r", "Refresh"},
	}

	for _, row := range sections {
		if len(row) < 2 || (row[0] == "" && row[1] == "") {
			sb.WriteString("\n")
			continue
		}
		if row[1] == "" {
			sb.WriteString(sHighlight.Render("  " + row[0]))
			sb.WriteString("\n")
			continue
		}
		key := sHighlight.Render(padRight(row[0], 18))
		sb.WriteString(fmt.Sprintf("    %s %s\n", key, row[1]))
	}

	sb.WriteString("\n")
	sb.WriteString(sMuted.Render("  Press q or Esc to close"))

	content := sb.String()
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cK9sPrimary).
		Padding(1, 2).
		Width(min(m.width-4, 60)).
		Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
