package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func renderAgentsTab(m model) string {
	cols := []TableColumn{
		{Title: "NODE ID", Width: 20, Align: lipgloss.Left},
		{Title: "STATUS", Width: 15, Align: lipgloss.Left},
		{Title: "VERSION", Width: 15, Align: lipgloss.Left},
		{Title: "LAST SEEN", Width: 20, Align: lipgloss.Left},
		{Title: "SERVICES", Width: 0, Align: lipgloss.Right, Flex: true},
	}

	rows := make([]TableRow, len(mockAgents))
	for i, a := range mockAgents {
		var statusColored, statusPlain string
		if a.status == "active" {
			statusColored = sSuccess.Render("active")
		} else {
			statusColored = sError.Render("offline")
		}
		statusPlain = a.status

		svcStr := fmt.Sprintf("%d", a.services)
		rows[i] = TableRow{
			Cells: []string{a.nodeID, statusColored, a.version, a.lastSeen, svcStr},
			Plain: []string{a.nodeID, statusPlain, a.version, a.lastSeen, svcStr},
		}
	}

	table := NewTable(cols)
	table.SetWidth(m.width)
	table.SetRows(rows)
	table.cursor = m.cursor
	return table.View()
}
