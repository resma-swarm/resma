package main

import (
	"fmt"
	"strings"
)

func renderAgentsTab(m model) string {
	var sb strings.Builder

	idW := 20
	statW := 15
	verW := 15
	seenW := 20
	svcsW := m.width - idW - statW - verW - seenW - 5
	if svcsW < 10 {
		svcsW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s",
		padRight(sK9sTableHeader.Render("NODE ID"), idW),
		padRight(sK9sTableHeader.Render("STATUS"), statW),
		padRight(sK9sTableHeader.Render("VERSION"), verW),
		padRight(sK9sTableHeader.Render("LAST SEEN"), seenW),
		sK9sTableHeader.Render("SERVICES"),
	)
	sb.WriteString(header + "\n")

	for i, a := range mockAgents {
		style := sK9sInfoVal
		if i == m.cursor {
			style = sK9sTableCursor
		}

		status := sK9sGreen.Render("active")
		if a.status != "active" {
			status = sK9sRed.Render("offline")
		}

		row := fmt.Sprintf("%s %s %s %s %s",
			padRight(a.nodeID, idW),
			padRight(status, statW),
			padRight(a.version, verW),
			padRight(a.lastSeen, seenW),
			padLeft(fmt.Sprintf("%d", a.services), svcsW),
		)
		sb.WriteString(style.Width(m.width).Render(row) + "\n")
	}

	return sb.String()
}
