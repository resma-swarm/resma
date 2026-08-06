package main

import (
	"fmt"
	"strings"
)

func renderServicesTab(m model) string {
	var sb strings.Builder

	// Calcula widths dinâmicos
	nameW := 25
	replW := 10
	cpuW := 10
	memW := 10
	statusW := 15
	trendW := m.width - nameW - replW - cpuW - memW - statusW - 5
	if trendW < 10 {
		trendW = 10
	}

	// Header
	header := fmt.Sprintf("%s %s %s %s %s %s",
		padRight(sK9sTableHeader.Render("NAME"), nameW),
		padLeft(sK9sTableHeader.Render("READY"), replW),
		padLeft(sK9sTableHeader.Render("CPU"), cpuW),
		padLeft(sK9sTableHeader.Render("MEM"), memW),
		padRight(sK9sTableHeader.Render("STATUS"), statusW),
		sK9sTableHeader.Render("TREND"),
	)
	sb.WriteString(header + "\n")

	for i, s := range mockServices {
		style := sK9sInfoVal
		if i == m.cursor {
			style = sK9sTableCursor
		}

		cpu := fmt.Sprintf("%.1f", s.cpu)
		mem := fmt.Sprintf("%.1f", s.mem)

		status := sK9sGreen.Render("running")
		if s.status != "running" {
			status = sK9sRed.Render("stopped")
		}

		row := fmt.Sprintf("%s %s %s %s %s %s",
			padRight(s.name, nameW),
			padLeft(s.replicas, replW),
			padLeft(cpu, cpuW),
			padLeft(mem, memW),
			padRight(status, statusW),
			sparkline(s.spark, trendW),
		)
		sb.WriteString(style.Width(m.width).Render(row) + "\n")
	}

	return sb.String()
}

var (
	sK9sGreen = sSuccess.Copy().Foreground(cK9sGreen)
	sK9sRed   = sError.Copy().Foreground(cK9sRed)
)
