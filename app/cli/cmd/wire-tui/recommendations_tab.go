package main

import (
	"fmt"
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
	reasW := m.width - svcW - riskW - tierW - cpuW - memW - 2
	if reasW < 10 {
		reasW = 10
	}

	header := fmt.Sprintf("%s %s %s %s %s %s",
		padRight(sTableHeader.Render("SERVICE"), svcW),
		padRight(sTableHeader.Render("RISK"), riskW),
		padRight(sTableHeader.Render("TIER"), tierW),
		padRight(sTableHeader.Render("CPU"), cpuW),
		padRight(sTableHeader.Render("MEM"), memW),
		sTableHeader.Render("REASON"),
	)
	sb.WriteString(header + "\n")

	for i, r := range mockRecs {
		if i == m.cursor {
			rawRow := fmt.Sprintf("%s %s %s %s %s %s",
				padRight(r.service, svcW),
				padRight(r.risk, riskW),
				padRight(r.tier, tierW),
				padRight(r.cpu, cpuW),
				padRight(r.mem, memW),
				truncate(r.reason, reasW),
			)
			fullRow := padRight(rawRow, m.width)
			sb.WriteString(sTableCursor.Render(fullRow) + "\n\n")
		} else {
			riskColored := sSuccess.Render("low")
			if r.risk == "high" {
				riskColored = sError.Render("high")
			} else if r.risk == "medium" {
				riskColored = sWarning.Render("medium")
			}

			row := fmt.Sprintf("%s %s %s %s %s %s",
				padRight(r.service, svcW),
				padRight(riskColored, riskW),
				padRight(r.tier, tierW),
				padRight(r.cpu, cpuW),
				padRight(r.mem, memW),
				truncate(r.reason, reasW),
			)
			sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(row) + "\n\n")
		}
	}

	return sb.String()
}
