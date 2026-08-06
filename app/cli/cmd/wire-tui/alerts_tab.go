package main

import (
	"fmt"
	"strings"
)

func renderAlertsTab(m model) string {
	var sb strings.Builder
	sb.WriteString(sTitle.Render("Alerts — 6 active (2 critical, 3 warning, 1 info)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s %s %s\n",
		padRight(sTableHeader.Render("TIME"), 10),
		padRight(sTableHeader.Render("LEVEL"), 8),
		padRight(sTableHeader.Render("SERVICE"), 20),
		sTableHeader.Render("MESSAGE"),
	))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 70)))
	sb.WriteString("\n")

	for i, a := range mockAlerts {
		var levelStyled string
		switch a.level {
		case "critical":
			levelStyled = sError.Render("CRIT")
		case "warning":
			levelStyled = sWarning.Render("WARN")
		default:
			levelStyled = sMuted.Render("INFO")
		}

		time := padRight(a.time, 10)
		if i == m.cursor {
			time = sSelected.Render(padRight(a.time, 10))
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s\n",
			time,
			padRight(levelStyled, 8),
			padRight(a.service, 20),
			truncate(a.message, 40),
		))
	}

	return sb.String()
}
