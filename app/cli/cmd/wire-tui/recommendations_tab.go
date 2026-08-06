package main

import (
	"fmt"
	"strings"
)

func renderRecommendationsTab(m model) string {
	var sb strings.Builder

	svcW := 25
	riskW := 10
	tierW := 15
	cpuW := 10
	memW := 10
	reasW := m.width - svcW - riskW - tierW - cpuW - memW - 5
	if reasW < 10 {
		reasW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s %s",
		padRight(sK9sTableHeader.Render("SERVICE"), svcW),
		padRight(sK9sTableHeader.Render("RISK"), riskW),
		padRight(sK9sTableHeader.Render("TIER"), tierW),
		padRight(sK9sTableHeader.Render("CPU"), cpuW),
		padRight(sK9sTableHeader.Render("MEM"), memW),
		sK9sTableHeader.Render("REASON"),
	)
	sb.WriteString(header + "\n")

	for i, r := range mockRecs {
		style := sK9sInfoVal
		if i == m.cursor {
			style = sK9sTableCursor
		}

		risk := sK9sGreen.Render("low")
		if r.risk == "high" {
			risk = sK9sRed.Render("high")
		} else if r.risk == "medium" {
			risk = sWarning.Render("medium")
		}

		row := fmt.Sprintf("%s %s %s %s %s %s",
			padRight(r.service, svcW),
			padRight(risk, riskW),
			padRight(r.tier, tierW),
			padRight(r.cpu, cpuW),
			padRight(r.mem, memW),
			truncate(r.reason, reasW),
		)
		sb.WriteString(style.Width(m.width).Render(row) + "\n")
	}

	return sb.String()
}
