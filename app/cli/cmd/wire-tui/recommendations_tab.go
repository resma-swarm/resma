package main

import (
	"fmt"
	"strings"
)

func renderRecommendationsTab(m model) string {
	var sb strings.Builder
	sb.WriteString(sTitle.Render("Recommendations — 6 pending (2 high, 2 medium, 2 low)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
		padRight(sTableHeader.Render("SERVICE"), 18),
		padRight(sTableHeader.Render("RISK"), 8),
		padRight(sTableHeader.Render("TIER"), 14),
		padRight(sTableHeader.Render("CPU"), 10),
		padRight(sTableHeader.Render("MEM"), 8),
		sTableHeader.Render("REASON"),
	))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 75)))
	sb.WriteString("\n")

	for i, r := range mockRecs {
		riskStyled := r.risk
		switch r.risk {
		case "high":
			riskStyled = sError.Render("high")
		case "medium":
			riskStyled = sWarning.Render("medium")
		default:
			riskStyled = sSuccess.Render("low")
		}

		tierStyled := r.tier
		switch r.tier {
		case "conservative":
			tierStyled = sMuted.Render("conservative")
		case "balanced":
			tierStyled = sHighlight.Render("balanced")
		case "aggressive":
			tierStyled = sWarning.Render("aggressive")
		}

		svc := padRight(r.service, 18)
		if i == m.cursor {
			svc = sSelected.Render(padRight(r.service, 18))
		}

		sb.WriteString(fmt.Sprintf("%s %s %s %s %s %s\n",
			svc,
			padRight(riskStyled, 8),
			padRight(tierStyled, 14),
			padRight(r.cpu, 10),
			padRight(r.mem, 8),
			truncate(r.reason, 35),
		))
	}

	sb.WriteString("\n")
	sb.WriteString(sMuted.Render("  Apply with: a  ·  resma recommendations apply <service> --confirm"))

	return sb.String()
}
