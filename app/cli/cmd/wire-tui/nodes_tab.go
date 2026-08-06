package main

import (
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
	spaces := 6
	statusW := colWidths(m.width, []int{idW, hostW, cpuW, memW, diskW, roleW}, spaces)

	header := joinRow(
		cellLeft(sTableHeader.Render("NAME"), idW),
		cellLeft(sTableHeader.Render("HOSTNAME"), hostW),
		cellRight(sTableHeader.Render("CPU%"), cpuW),
		cellRight(sTableHeader.Render("MEM%"), memW),
		cellRight(sTableHeader.Render("DISK%"), diskW),
		cellLeft(sTableHeader.Render("ROLE"), roleW),
		cellLeft(sTableHeader.Render("STATUS"), statusW),
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(header) + "\n")

	for i, n := range mockNodes {
		statusStr := n.status

		var statusCell string
		if i == m.cursor {
			statusCell = statusStr
		} else if n.status == "ready" {
			statusCell = sSuccess.Render("ready")
		} else {
			statusCell = sError.Render("down")
		}

		cells := []string{
			cellLeft(n.id, idW),
			cellLeft(n.hostname, hostW),
			cellRight(pctStr(n.cpu), cpuW),
			cellRight(pctStr(n.mem), memW),
			cellRight(pctStr(n.disk), diskW),
			cellLeft(n.role, roleW),
			cellLeft(statusCell, statusW),
		}
		sb.WriteString(renderTableRow(cells, m.width, i == m.cursor) + "\n")
	}

	return sb.String()
}
