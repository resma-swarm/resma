package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderServicesTab(m model) string {
	var sb strings.Builder

	nameW := 25
	replW := 10
	cpuW := 10
	memW := 10
	statusW := 15
	spaces := 5 // 5 espaços entre 6 colunas
	trendW := colWidths(m.width, []int{nameW, replW, cpuW, memW, statusW}, spaces)

	// Header
	header := joinRow(
		cellLeft(sTableHeader.Render("NAME"), nameW),
		cellRight(sTableHeader.Render("READY"), replW),
		cellRight(sTableHeader.Render("CPU"), cpuW),
		cellRight(sTableHeader.Render("MEM"), memW),
		cellLeft(sTableHeader.Render("STATUS"), statusW),
		cellLeft(sTableHeader.Render("TREND"), trendW),
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(header) + "\n")

	for i, s := range mockServices {
		cpu := pctStr(s.cpu)
		mem := pctStr(s.mem)
		statusStr := "running"
		if s.status != "running" {
			statusStr = "stopped"
		}

		var spark string
		var statusCell string

		if i == m.cursor {
			// Linha selecionada: texto plain, cursor background cuida das cores
			spark = sparklinePlain(s.spark)
			statusCell = statusStr
		} else {
			spark = sparkline(s.spark, trendW)
			if s.status == "running" {
				statusCell = sSuccess.Render("running")
			} else {
				statusCell = sError.Render("stopped")
			}
		}

		cells := []string{
			cellLeft(s.name, nameW),
			cellRight(s.replicas, replW),
			cellRight(cpu, cpuW),
			cellRight(mem, memW),
			cellLeft(statusCell, statusW),
			cellLeft(spark, trendW),
		}
		sb.WriteString(renderTableRow(cells, m.width, i == m.cursor) + "\n")
	}

	return sb.String()
}
