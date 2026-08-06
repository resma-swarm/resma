package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderAgentsTab(m model) string {
	var sb strings.Builder

	idW := 20
	statW := 15
	verW := 15
	seenW := 20
	spaces := 4
	svcsW := colWidths(m.width, []int{idW, statW, verW, seenW}, spaces)

	header := joinRow(
		cellLeft(sTableHeader.Render("NODE ID"), idW),
		cellLeft(sTableHeader.Render("STATUS"), statW),
		cellLeft(sTableHeader.Render("VERSION"), verW),
		cellLeft(sTableHeader.Render("LAST SEEN"), seenW),
		cellRight(sTableHeader.Render("SERVICES"), svcsW),
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(header) + "\n")

	for i, a := range mockAgents {
		var statusCell string
		if i == m.cursor {
			statusCell = a.status
		} else if a.status == "active" {
			statusCell = sSuccess.Render("active")
		} else {
			statusCell = sError.Render("offline")
		}

		cells := []string{
			cellLeft(a.nodeID, idW),
			cellLeft(statusCell, statW),
			cellLeft(a.version, verW),
			cellLeft(a.lastSeen, seenW),
			cellRight(fmt.Sprintf("%d", a.services), svcsW),
		}
		sb.WriteString(renderTableRow(cells, m.width, i == m.cursor) + "\n")
	}

	return sb.String()
}
