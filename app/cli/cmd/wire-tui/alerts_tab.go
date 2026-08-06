package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderAlertsTab(m model) string {
	var sb strings.Builder

	timeW := 15
	levW := 10
	svcW := 25
	spaces := 3
	msgW := colWidths(m.width, []int{timeW, levW, svcW}, spaces)

	header := joinRow(
		cellLeft(sTableHeader.Render("TIME"), timeW),
		cellLeft(sTableHeader.Render("LEVEL"), levW),
		cellLeft(sTableHeader.Render("SERVICE"), svcW),
		cellLeft(sTableHeader.Render("MESSAGE"), msgW),
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(header) + "\n")

	for i, a := range mockAlerts {
		var levelCell string
		if i == m.cursor {
			levelCell = a.level
		} else {
			switch a.level {
			case "critical":
				levelCell = sError.Render("critical")
			case "warning":
				levelCell = sWarning.Render("warning")
			default:
				levelCell = sMuted.Render("info")
			}
		}

		cells := []string{
			cellLeft(a.time, timeW),
			cellLeft(levelCell, levW),
			cellLeft(a.service, svcW),
			cellLeft(truncate(a.message, msgW), msgW),
		}
		sb.WriteString(renderTableRow(cells, m.width, i == m.cursor) + "\n")
	}

	return sb.String()
}
