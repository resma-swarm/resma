package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderNodesTab(m model) string {
	var sb strings.Builder

	idW := 15
	hostW := 25
	cpuW := 10
	memW := 10
	diskW := 10
	roleW := 10
	statusW := m.width - idW - hostW - cpuW - memW - diskW - roleW - 2
	if statusW < 10 {
		statusW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s %s %s",
		padRight(sTableHeader.Render("NAME"), idW),
		padRight(sTableHeader.Render("HOSTNAME"), hostW),
		padLeft(sTableHeader.Render("CPU%"), cpuW),
		padLeft(sTableHeader.Render("MEM%"), memW),
		padLeft(sTableHeader.Render("DISK%"), diskW),
		padRight(sTableHeader.Render("ROLE"), roleW),
		sTableHeader.Render("STATUS"),
	)
	sb.WriteString(header + "\n")

	for i, n := range mockNodes {
		statusStr := "ready"
		if n.status != "ready" {
			statusStr = "down"
		}

		if i == m.cursor {
			rawRow := fmt.Sprintf("%s %s %s %s %s %s %s",
				padRight(n.id, idW),
				padRight(n.hostname, hostW),
				padLeft(fmt.Sprintf("%.1f%%", n.cpu), cpuW),
				padLeft(fmt.Sprintf("%.1f%%", n.mem), memW),
				padLeft(fmt.Sprintf("%.1f%%", n.disk), diskW),
				padRight(n.role, roleW),
				statusStr,
			)
			fullRow := padRight(rawRow, m.width)
			sb.WriteString(sTableCursor.Render(fullRow) + "\n\n")
		} else {
			statusColored := sSuccess.Render("ready")
			if n.status != "ready" {
				statusColored = sError.Render("down")
			}

			row := fmt.Sprintf("%s %s %s %s %s %s %s",
				padRight(n.id, idW),
				padRight(n.hostname, hostW),
				padLeft(fmt.Sprintf("%.1f%%", n.cpu), cpuW),
				padLeft(fmt.Sprintf("%.1f%%", n.mem), memW),
				padLeft(fmt.Sprintf("%.1f%%", n.disk), diskW),
				padRight(n.role, roleW),
				statusColored,
			)
			sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(row) + "\n\n")
		}
	}

	return sb.String()
}
