package main

import (
	"github.com/charmbracelet/lipgloss"
)

func renderTasksTab(m model) string {
	cols := []TableColumn{
		{Title: "NAME", Width: 15, Align: lipgloss.Left},
		{Title: "SERVICE", Width: 25, Align: lipgloss.Left},
		{Title: "NODE", Width: 15, Align: lipgloss.Left},
		{Title: "STATUS", Width: 15, Align: lipgloss.Left},
		{Title: "DESIRED", Width: 15, Align: lipgloss.Left},
		{Title: "UPTIME", Width: 0, Align: lipgloss.Right, Flex: true},
	}

	rows := make([]TableRow, len(mockTasks))
	for i, t := range mockTasks {
		var statusColored, statusPlain string
		if t.status == "running" {
			statusColored = sSuccess.Render("running")
		} else {
			statusColored = sError.Render("failed")
		}
		statusPlain = t.status

		rows[i] = TableRow{
			Cells: []string{t.id, t.service, t.node, statusColored, t.desired, t.uptime},
			Plain: []string{t.id, t.service, t.node, statusPlain, t.desired, t.uptime},
		}
	}

	table := NewTable(cols)
	table.SetWidth(m.width)
	table.SetRows(rows)
	table.cursor = m.cursor
	applySortState(&table, m)
	return table.View()
}
