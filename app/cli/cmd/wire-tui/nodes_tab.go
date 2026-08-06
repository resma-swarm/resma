package main

import (
	"fmt"
	"strings"
)

func renderNodesTab(m model) string {
	var sb strings.Builder
	sb.WriteString(sTitle.Render("Nodes — 5 total (4 ready, 1 down)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s %s\n",
		padRight(sTableHeader.Render("ID"), 12),
		padRight(sTableHeader.Render("HOSTNAME"), 22),
		padLeft(sTableHeader.Render("CPU%"), 8),
		padLeft(sTableHeader.Render("MEM%"), 8),
		padLeft(sTableHeader.Render("DISK%"), 8),
		padRight(sTableHeader.Render("ROLE"), 8),
		sTableHeader.Render("STATUS"),
	))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 78)))
	sb.WriteString("\n")

	for i, n := range mockNodes {
		cpu := fmt.Sprintf("%.1f", n.cpu)
		mem := fmt.Sprintf("%.1f", n.mem)
		disk := fmt.Sprintf("%.1f", n.disk)

		cpuStyled := cpu
		if n.cpu > 80 {
			cpuStyled = sError.Render(cpu)
		} else if n.cpu > 60 {
			cpuStyled = sWarning.Render(cpu)
		}

		diskStyled := disk
		if n.disk > 60 {
			diskStyled = sWarning.Render(disk)
		}

		status := sSuccess.Render("ready")
		if n.status != "ready" {
			status = sError.Render("down")
		}

		id := padRight(n.id, 12)
		if i == m.cursor {
			id = sSelected.Render(padRight(n.id, 12))
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s %s\n",
			id,
			padRight(n.hostname, 22),
			padLeft(cpuStyled, 8),
			padLeft(mem, 8),
			padLeft(diskStyled, 8),
			padRight(n.role, 8),
			status,
		))
	}

	return sb.String()
}
