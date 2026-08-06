package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderTasksTab(m model) string {
	var sb strings.Builder

	idW := 15
	svcW := 25
	nodeW := 15
	statW := 15
	desW := 15
	upW := m.width - idW - svcW - nodeW - statW - desW - 2
	if upW < 10 {
		upW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s %s",
		padRight(sTableHeader.Render("NAME"), idW),
		padRight(sTableHeader.Render("SERVICE"), svcW),
		padRight(sTableHeader.Render("NODE"), nodeW),
		padRight(sTableHeader.Render("STATUS"), statW),
		padRight(sTableHeader.Render("DESIRED"), desW),
		sTableHeader.Render("UPTIME"),
	)
	sb.WriteString(header + "\n")

	for i, t := range mockTasks {
		statusStr := t.status

		if i == m.cursor {
			rawRow := fmt.Sprintf("%s %s %s %s %s %s",
				padRight(t.id, idW),
				padRight(t.service, svcW),
				padRight(t.node, nodeW),
				padRight(statusStr, statW),
				padRight(t.desired, desW),
				padLeft(t.uptime, upW),
			)
			fullRow := padRight(rawRow, m.width)
			sb.WriteString(sTableCursor.Render(fullRow) + "\n\n")
		} else {
			statusColored := sSuccess.Render("running")
			if t.status != "running" {
				statusColored = sError.Render("failed")
			}

			row := fmt.Sprintf("%s %s %s %s %s %s",
				padRight(t.id, idW),
				padRight(t.service, svcW),
				padRight(t.node, nodeW),
				padRight(statusColored, statW),
				padRight(t.desired, desW),
				padLeft(t.uptime, upW),
			)
			sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(row) + "\n\n")
		}
	}

	return sb.String()
}
