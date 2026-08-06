package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderRecommendationsTab(m model) string {
	var sb strings.Builder

	svcW := 25
	riskW := 10
	tierW := 15
	cpuW := 10
	memW := 10
	spaces := 5
	reasW := colWidths(m.width, []int{svcW, riskW, tierW, cpuW, memW}, spaces)

	header := joinRow(
		cellLeft(sTableHeader.Render("SERVICE"), svcW),
		cellLeft(sTableHeader.Render("RISK"), riskW),
		cellLeft(sTableHeader.Render("TIER"), tierW),
		cellLeft(sTableHeader.Render("CPU"), cpuW),
		cellLeft(sTableHeader.Render("MEM"), memW),
		cellLeft(sTableHeader.Render("REASON"), reasW),
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(header) + "\n")

	for i, r := range mockRecs {
		var riskCell string
		if i == m.cursor {
			riskCell = r.risk
		} else if r.risk == "high" {
			riskCell = sError.Render("high")
		} else if r.risk == "medium" {
			riskCell = sWarning.Render("medium")
		} else {
			riskCell = sSuccess.Render("low")
		}

		cells := []string{
			cellLeft(r.service, svcW),
			cellLeft(riskCell, riskW),
			cellLeft(r.tier, tierW),
			cellLeft(r.cpu, cpuW),
			cellLeft(r.mem, memW),
			cellLeft(truncate(r.reason, reasW), reasW),
		}
		sb.WriteString(renderTableRow(cells, m.width, i == m.cursor) + "\n")
	}

	return sb.String()
}
