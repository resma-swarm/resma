package main

import (
	"github.com/charmbracelet/lipgloss"
)

func renderNodesTab(m model) string {
	cols := []TableColumn{
		{Title: "NAME", Width: 15, Align: lipgloss.Left},
		{Title: "HOSTNAME", Width: 25, Align: lipgloss.Left},
		{Title: "CPU%", Width: 10, Align: lipgloss.Right},
		{Title: "MEM%", Width: 10, Align: lipgloss.Right},
		{Title: "DISK%", Width: 10, Align: lipgloss.Right},
		{Title: "ROLE", Width: 10, Align: lipgloss.Left},
		{Title: "STATUS", Width: 0, Align: lipgloss.Left, Flex: true},
	}

	rows := make([]TableRow, len(mockNodes))
	for i, n := range mockNodes {
		var statusColored, statusPlain string
		if n.status == "ready" {
			statusColored = sSuccess.Render("ready")
		} else {
			statusColored = sError.Render("down")
		}
		statusPlain = n.status

		rows[i] = TableRow{
			Cells: []string{
				n.id, n.hostname, pctStr(n.cpu), pctStr(n.mem), pctStr(n.disk), n.role, statusColored,
			},
			Plain: []string{
				n.id, n.hostname, pctStr(n.cpu), pctStr(n.mem), pctStr(n.disk), n.role, statusPlain,
			},
		}
	}

	table := NewTable(cols)
	table.SetWidth(m.width)
	table.SetRows(rows)
	table.cursor = m.cursor
	return table.View()
}
