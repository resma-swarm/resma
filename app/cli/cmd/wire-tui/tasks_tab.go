package main

import (
	"fmt"
	"strings"
)

func renderTasksTab(m model) string {
	var sb strings.Builder

	idW := 15
	svcW := 25
	nodeW := 15
	statW := 15
	desW := 15
	upW := m.width - idW - svcW - nodeW - statW - desW - 5
	if upW < 10 {
		upW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s %s",
		padRight(sK9sTableHeader.Render("NAME"), idW),
		padRight(sK9sTableHeader.Render("SERVICE"), svcW),
		padRight(sK9sTableHeader.Render("NODE"), nodeW),
		padRight(sK9sTableHeader.Render("STATUS"), statW),
		padRight(sK9sTableHeader.Render("DESIRED"), desW),
		sK9sTableHeader.Render("UPTIME"),
	)
	sb.WriteString(header + "\n")

	for i, t := range mockTasks {
		style := sK9sInfoVal
		if i == m.cursor {
			style = sK9sTableCursor
		}

		status := sK9sGreen.Render("running")
		if t.status != "running" {
			status = sK9sRed.Render("failed")
		}

		row := fmt.Sprintf("%s %s %s %s %s %s",
			padRight(t.id, idW),
			padRight(t.service, svcW),
			padRight(t.node, nodeW),
			padRight(status, statW),
			padRight(t.desired, desW),
			padLeft(t.uptime, upW),
		)
		sb.WriteString(style.Width(m.width).Render(row) + "\n")
	}

	return sb.String()
}
