package main

import (
	"github.com/charmbracelet/lipgloss"
)

func renderAlertsTab(m model) string {
	cols := []TableColumn{
		{Title: "TIME", Width: 15, Align: lipgloss.Left},
		{Title: "LEVEL", Width: 10, Align: lipgloss.Left},
		{Title: "SERVICE", Width: 25, Align: lipgloss.Left},
		{Title: "MESSAGE", Width: 0, Align: lipgloss.Left, Flex: true},
	}

	rows := make([]TableRow, len(mockAlerts))
	for i, a := range mockAlerts {
		var levelColored, levelPlain string
		switch a.level {
		case "critical":
			levelColored = sError.Render("critical")
			levelPlain = "critical"
		case "warning":
			levelColored = sWarning.Render("warning")
			levelPlain = "warning"
		default:
			levelColored = sMuted.Render("info")
			levelPlain = "info"
		}

		rows[i] = TableRow{
			Cells: []string{a.time, levelColored, a.service, a.message},
			Plain: []string{a.time, levelPlain, a.service, a.message},
		}
	}

	table := NewTable(cols)
	table.SetWidth(m.width)
	table.SetRows(rows)
	table.cursor = m.cursor
	return table.View()
}
