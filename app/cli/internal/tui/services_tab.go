package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func renderServicesTab(m model) string {
	cols := []TableColumn{
		{Title: "NAME", Width: 25, Align: lipgloss.Left},
		{Title: "READY", Width: 10, Align: lipgloss.Right},
		{Title: "CPU", Width: 10, Align: lipgloss.Right},
		{Title: "MEM", Width: 10, Align: lipgloss.Right},
		{Title: "STATUS", Width: 15, Align: lipgloss.Left},
		{Title: "TREND", Width: 0, Align: lipgloss.Left, Flex: true},
	}

	rows := make([]TableRow, len(mockServices))
	for i, s := range mockServices {
		cpu := pctStr(s.cpu)
		mem := pctStr(s.mem)
		statusStr := "running"
		if s.status != "running" {
			statusStr = "stopped"
		}

		var sparkColored, sparkPlainStr string
		var statusColored, statusPlain string

		sparkColored = sparkline(s.spark, 30)
		sparkPlainStr = brailleSparklinePlain(s.spark, 30)

		if s.status == "running" {
			statusColored = sSuccess.Render("running")
		} else {
			statusColored = sError.Render("stopped")
		}
		statusPlain = statusStr

		rows[i] = TableRow{
			Cells: []string{
				s.name, s.replicas, cpu, mem, statusColored, sparkColored,
			},
			Plain: []string{
				s.name, s.replicas, cpu, mem, statusPlain, sparkPlainStr,
			},
		}
	}

	table := NewTable(cols)
	table.SetWidth(m.width)
	table.SetRows(rows)
	table.cursor = m.cursor
	applySortState(&table, m)
	return table.View()
}
