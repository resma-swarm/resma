package main

import (
	"fmt"
	"strings"
)

func renderServicesTab(m model) string {
	var sb strings.Builder
	sb.WriteString(sTitle.Render("Services — 8 total (7 running, 1 stopped)"))
	sb.WriteString("\n\n")

	// Table header
	sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
		padRight(sTableHeader.Render("NAME"), 20),
		padLeft(sTableHeader.Render("REPLICAS"), 10),
		padLeft(sTableHeader.Render("CPU%"), 8),
		padLeft(sTableHeader.Render("MEM%"), 8),
		padRight(sTableHeader.Render("STATUS"), 12),
		sTableHeader.Render("TREND"),
	))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 78)))
	sb.WriteString("\n")

	for i, s := range mockServices {
		cpu := fmt.Sprintf("%.1f", s.cpu)
		mem := fmt.Sprintf("%.1f", s.mem)

		cpuStyled := cpu
		if s.cpu > 70 {
			cpuStyled = sError.Render(cpu)
		} else if s.cpu > 50 {
			cpuStyled = sWarning.Render(cpu)
		}

		memStyled := mem
		if s.mem > 80 {
			memStyled = sError.Render(mem)
		} else if s.mem > 60 {
			memStyled = sWarning.Render(mem)
		}

		status := sSuccess.Render("running")
		if s.status != "running" {
			status = sError.Render("stopped")
		}

		name := padRight(s.name, 20)
		if i == m.cursor {
			name = sSelected.Render(padRight(s.name, 20))
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s %s  %s\n",
			name,
			padLeft(s.replicas, 10),
			padLeft(cpuStyled, 8),
			padLeft(memStyled, 8),
			padRight(status, 12),
			sparkline(s.spark, 20),
		))
	}

	return sb.String()
}
