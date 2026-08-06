package main

import (
	"github.com/charmbracelet/lipgloss"
)

func renderRecommendationsTab(m model) string {
	cols := []TableColumn{
		{Title: "SERVICE", Width: 25, Align: lipgloss.Left},
		{Title: "RISK", Width: 10, Align: lipgloss.Left},
		{Title: "TIER", Width: 15, Align: lipgloss.Left},
		{Title: "CPU", Width: 10, Align: lipgloss.Left},
		{Title: "MEM", Width: 10, Align: lipgloss.Left},
		{Title: "REASON", Width: 0, Align: lipgloss.Left, Flex: true},
	}

	rows := make([]TableRow, len(mockRecs))
	for i, r := range mockRecs {
		var riskColored, riskPlain string
		switch r.risk {
		case "high":
			riskColored = sError.Render("high")
			riskPlain = "high"
		case "medium":
			riskColored = sWarning.Render("medium")
			riskPlain = "medium"
		default:
			riskColored = sSuccess.Render("low")
			riskPlain = "low"
		}

		rows[i] = TableRow{
			Cells: []string{r.service, riskColored, r.tier, r.cpu, r.mem, r.reason},
			Plain: []string{r.service, riskPlain, r.tier, r.cpu, r.mem, r.reason},
		}
	}

	table := NewTable(cols)
	table.SetWidth(m.width)
	table.SetRows(rows)
	table.cursor = m.cursor
	return table.View()
}
