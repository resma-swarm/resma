package main

import (
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
	spaces := 5
	upW := colWidths(m.width, []int{idW, svcW, nodeW, statW, desW}, spaces)

	header := joinRow(
		cellLeft(sTableHeader.Render("NAME"), idW),
		cellLeft(sTableHeader.Render("SERVICE"), svcW),
		cellLeft(sTableHeader.Render("NODE"), nodeW),
		cellLeft(sTableHeader.Render("STATUS"), statW),
		cellLeft(sTableHeader.Render("DESIRED"), desW),
		cellRight(sTableHeader.Render("UPTIME"), upW),
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(header) + "\n")

	for i, t := range mockTasks {
		var statusCell string
		if i == m.cursor {
			statusCell = t.status
		} else if t.status == "running" {
			statusCell = sSuccess.Render("running")
		} else {
			statusCell = sError.Render("failed")
		}

		cells := []string{
			cellLeft(t.id, idW),
			cellLeft(t.service, svcW),
			cellLeft(t.node, nodeW),
			cellLeft(statusCell, statW),
			cellLeft(t.desired, desW),
			cellRight(t.uptime, upW),
		}
		sb.WriteString(renderTableRow(cells, m.width, i == m.cursor) + "\n")
	}

	return sb.String()
}
