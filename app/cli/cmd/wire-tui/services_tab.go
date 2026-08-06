package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderServicesTab(m model) string {
	var sb strings.Builder

	// Calcula widths dinâmicos de colunas
	nameW := 25
	replW := 10
	cpuW := 10
	memW := 10
	statusW := 15
	trendW := m.width - nameW - replW - cpuW - memW - statusW - 2
	if trendW < 10 {
		trendW = 10
	}

	// Header da tabela
	header := fmt.Sprintf("%s %s %s %s %s %s",
		padRight(sTableHeader.Render("NAME"), nameW),
		padLeft(sTableHeader.Render("READY"), replW),
		padLeft(sTableHeader.Render("CPU"), cpuW),
		padLeft(sTableHeader.Render("MEM"), memW),
		padRight(sTableHeader.Render("STATUS"), statusW),
		sTableHeader.Render("TREND"),
	)
	sb.WriteString(header + "\n")

	for i, s := range mockServices {
		cpu := fmt.Sprintf("%.1f%%", s.cpu)
		mem := fmt.Sprintf("%.1f%%", s.mem)

		statusStr := "running"
		if s.status != "running" {
			statusStr = "stopped"
		}

		// Se a linha estive selecionada, formatamos sem ANSI escuros prévios para pintar 100% da linha
		if i == m.cursor {
			rawRow := fmt.Sprintf("%s %s %s %s %s %s",
				padRight(s.name, nameW),
				padLeft(s.replicas, replW),
				padLeft(cpu, cpuW),
				padLeft(mem, memW),
				padRight(statusStr, statusW),
				sparkline(s.spark, trendW),
			)
			// Garante preenchimento de 100% da largura da tela com fundo Indigo
			fullRow := padRight(rawRow, m.width)
			sb.WriteString(sTableCursor.Render(fullRow) + "\n\n") // Adiciona espaçamento vertical
		} else {
			// Cores dinâmicas normais sem seleção
			statusColored := sSuccess.Render("running")
			if s.status != "running" {
				statusColored = sError.Render("stopped")
			}

			row := fmt.Sprintf("%s %s %s %s %s %s",
				padRight(s.name, nameW),
				padLeft(s.replicas, replW),
				padLeft(cpu, cpuW),
				padLeft(mem, memW),
				padRight(statusColored, statusW),
				sparkline(s.spark, trendW),
			)
			sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(row) + "\n\n") // Espaçamento vertical
		}
	}

	return sb.String()
}
