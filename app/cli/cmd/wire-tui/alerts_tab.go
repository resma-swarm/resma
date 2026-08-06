package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderAlertsTab(m model) string {
	var sb strings.Builder

	timeW := 15
	levW := 10
	svcW := 25
	msgW := m.width - timeW - levW - svcW - 2
	if msgW < 10 {
		msgW = 10
	}

	header := fmt.Sprintf("%s %s %s %s",
		padRight(sTableHeader.Render("TIME"), timeW),
		padRight(sTableHeader.Render("LEVEL"), levW),
		padRight(sTableHeader.Render("SERVICE"), svcW),
		sTableHeader.Render("MESSAGE"),
	)
	sb.WriteString(header + "\n")

	for i, a := range mockAlerts {
		if i == m.cursor {
			rawRow := fmt.Sprintf("%s %s %s %s",
				padRight(a.time, timeW),
				padRight(a.level, levW),
				padRight(a.service, svcW),
				truncate(a.message, msgW),
			)
			fullRow := padRight(rawRow, m.width)
			sb.WriteString(sTableCursor.Render(fullRow) + "\n\n")
		} else {
			var levelColored string
			switch a.level {
			case "critical":
				levelColored = sError.Render("CRIT")
			case "warning":
				levelColored = sWarning.Render("WARN")
			default:
				levelColored = sMuted.Render("INFO")
			}

			row := fmt.Sprintf("%s %s %s %s",
				padRight(a.time, timeW),
				padRight(levelColored, levW),
				padRight(a.service, svcW),
				truncate(a.message, msgW),
			)
			sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(row) + "\n\n")
		}
	}

	return sb.String()
}
