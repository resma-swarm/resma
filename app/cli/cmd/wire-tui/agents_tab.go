package main

import (
	"fmt"
	"strings"
)

func renderAgentsTab(m model) string {
	var sb strings.Builder
	sb.WriteString(sTitle.Render("Agents — 5 registered (4 active, 1 offline)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s %s %s %s\n",
		padRight(sTableHeader.Render("NODE ID"), 12),
		padRight(sTableHeader.Render("STATUS"), 10),
		padRight(sTableHeader.Render("VERSION"), 10),
		padRight(sTableHeader.Render("LAST SEEN"), 14),
		sTableHeader.Render("SERVICES"),
	))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 60)))
	sb.WriteString("\n")

	for i, a := range mockAgents {
		status := sSuccess.Render("active")
		if a.status != "active" {
			status = sError.Render("offline")
		}

		id := padRight(a.nodeID, 12)
		if i == m.cursor {
			id = sSelected.Render(padRight(a.nodeID, 12))
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s %s\n",
			id,
			padRight(status, 10),
			padRight(a.version, 10),
			padRight(a.lastSeen, 14),
			padLeft(fmt.Sprintf("%d", a.services), 8),
		))
	}

	return sb.String()
}
