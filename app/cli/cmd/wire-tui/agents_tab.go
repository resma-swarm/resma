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
	svcsW := m.width - idW - statW - verW - seenW - 2
	if svcsW < 10 {
		svcsW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s",
		padRight(sTableHeader.Render("NODE ID"), idW),
		padRight(sTableHeader.Render("STATUS"), statW),
		padRight(sTableHeader.Render("VERSION"), verW),
		padRight(sTableHeader.Render("LAST SEEN"), seenW),
		sTableHeader.Render("SERVICES"),
	)
	sb.WriteString(header + "\n")

	for i, a := range mockAgents {
		statusStr := a.status

		if i == m.cursor {
			rawRow := fmt.Sprintf("%s %s %s %s %s",
				padRight(a.nodeID, idW),
				padRight(statusStr, statW),
				padRight(a.version, verW),
				padRight(a.lastSeen, seenW),
				padLeft(fmt.Sprintf("%d", a.services), svcsW),
			)
			fullRow := padRight(rawRow, m.width)
			sb.WriteString(sTableCursor.Render(fullRow) + "\n\n")
		} else {
			statusColored := sSuccess.Render("active")
			if a.status != "active" {
				statusColored = sError.Render("offline")
			}

			row := fmt.Sprintf("%s %s %s %s %s",
				padRight(a.nodeID, idW),
				padRight(statusColored, statW),
				padRight(a.version, verW),
				padRight(a.lastSeen, seenW),
				padLeft(fmt.Sprintf("%d", a.services), svcsW),
			)
			sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(row) + "\n\n")
		}
	}

	return sb.String()
}
