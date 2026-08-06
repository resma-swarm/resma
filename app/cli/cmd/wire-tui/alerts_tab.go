package main

import (
	"fmt"
	"strings"
)

func renderAlertsTab(m model) string {
	var sb strings.Builder

	timeW := 15
	levW := 10
	svcW := 25
	msgW := m.width - timeW - levW - svcW - 5
	if msgW < 10 {
		msgW = 10
	}

	header := fmt.Sprintf("%s %s %s %s",
		padRight(sK9sTableHeader.Render("TIME"), timeW),
		padRight(sK9sTableHeader.Render("LEVEL"), levW),
		padRight(sK9sTableHeader.Render("SERVICE"), svcW),
		sK9sTableHeader.Render("MESSAGE"),
	)
	sb.WriteString(header + "\n")

	for i, a := range mockAlerts {
		style := sK9sInfoVal
		if i == m.cursor {
			style = sK9sTableCursor
		}

		var level string
		switch a.level {
		case "critical":
			level = sK9sRed.Render("CRIT")
		case "warning":
			level = sWarning.Render("WARN")
		default:
			level = sMuted.Render("INFO")
		}

		row := fmt.Sprintf("%s %s %s %s",
			padRight(a.time, timeW),
			padRight(level, levW),
			padRight(a.service, svcW),
			truncate(a.message, msgW),
		)
		sb.WriteString(style.Width(m.width).Render(row) + "\n")
	}

	return sb.String()
}
