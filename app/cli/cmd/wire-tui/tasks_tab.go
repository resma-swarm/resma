package main

import (
	"fmt"
	"strings"
)

func renderTasksTab(m model) string {
	var sb strings.Builder
	sb.WriteString(sTitle.Render("Tasks — 12 total (10 running, 2 failed)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
		padRight(sTableHeader.Render("ID"), 12),
		padRight(sTableHeader.Render("SERVICE"), 20),
		padRight(sTableHeader.Render("NODE"), 10),
		padRight(sTableHeader.Render("STATUS"), 10),
		padRight(sTableHeader.Render("DESIRED"), 10),
		sTableHeader.Render("UPTIME"),
	))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 72)))
	sb.WriteString("\n")

	for i, t := range mockTasks {
		status := sSuccess.Render("running")
		if t.status != "running" {
			status = sError.Render("failed")
		}

		id := padRight(t.id, 12)
		if i == m.cursor {
			id = sSelected.Render(padRight(t.id, 12))
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
			id,
			padRight(t.service, 20),
			padRight(t.node, 10),
			padRight(status, 10),
			padRight(t.desired, 10),
			padLeft(t.uptime, 10),
		))
	}

	return sb.String()
}
