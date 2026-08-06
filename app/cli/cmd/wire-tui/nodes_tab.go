package main

import (
	"fmt"
	"strings"
)

func renderNodesTab(m model) string {
	var sb strings.Builder

	idW := 15
	hostW := 25
	cpuW := 10
	memW := 10
	diskW := 10
	roleW := 10
	statusW := m.width - idW - hostW - cpuW - memW - diskW - roleW - 5
	if statusW < 10 {
		statusW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s %s %s",
		padRight(sK9sTableHeader.Render("NAME"), idW),
		padRight(sK9sTableHeader.Render("HOSTNAME"), hostW),
		padLeft(sK9sTableHeader.Render("CPU%"), cpuW),
		padLeft(sK9sTableHeader.Render("MEM%"), memW),
		padLeft(sK9sTableHeader.Render("DISK%"), diskW),
		padRight(sK9sTableHeader.Render("ROLE"), roleW),
		sK9sTableHeader.Render("STATUS"),
	)
	sb.WriteString(header + "\n")

	for i, n := range mockNodes {
		style := sK9sInfoVal
		if i == m.cursor {
			style = sK9sTableCursor
		}

		status := sK9sGreen.Render("ready")
		if n.status != "ready" {
			status = sK9sRed.Render("down")
		}

		row := fmt.Sprintf("%s %s %s %s %s %s %s",
			padRight(n.id, idW),
			padRight(n.hostname, hostW),
			padLeft(fmt.Sprintf("%.1f", n.cpu), cpuW),
			padLeft(fmt.Sprintf("%.1f", n.mem), memW),
			padLeft(fmt.Sprintf("%.1f", n.disk), diskW),
			padRight(n.role, roleW),
			status,
		)
		sb.WriteString(style.Width(m.width).Render(row) + "\n")
	}

	return sb.String()
}
