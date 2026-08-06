package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Logo ASCII do RESMA (6 linhas, estilo k9s)
var logoLines = []string{
	" ____  ____  ____  _  _ ",
	"|  _ \\|  _ \\/ ___|| || |",
	"| |_) | |_) \\___ \\| || |",
	"|  _ <|  __/ ___) |__   |",
	"|_| \\_\\_|  |____/   |_/ |",
	"                        ",
}

// renderLogo renderiza o logo ASCII + status bar.
func renderLogo(m model) string {
	var sb strings.Builder
	for _, line := range logoLines {
		sb.WriteString(sLogo.Render(line))
		sb.WriteString("\n")
	}
	// Status bar (1 linha) — cor muda com estado
	statusBar := renderStatusBar(m)
	sb.WriteString(statusBar)
	return sb.String()
}

func renderStatusBar(m model) string {
	var text, bg string
	switch {
	case m.viewMode == ViewCommand:
		text = " COMMAND "
		bg = sStatusCmd.Render(text)
	case m.viewMode == ViewFilter:
		text = " FILTER "
		bg = sStatusFilter.Render(text)
	case m.viewMode == ViewHelp:
		text = " HELP "
		bg = sStatusHelp.Render(text)
	case m.viewMode == ViewDetail:
		text = " DETAIL "
		bg = sStatusDetail.Render(text)
	default:
		text = " ONLINE "
		bg = sStatusOnline.Render(text)
	}
	return lipgloss.NewStyle().AlignHorizontal(lipgloss.Center).Render(bg)
}

// renderClusterInfo mostra info do cluster à esquerda do header.
func renderClusterInfo(m model) string {
	var sb strings.Builder
	sb.WriteString(sClusterTitle.Render(" RESMA "))
	sb.WriteString("\n")
	sb.WriteString(sMuted.Render(" Docker Swarm"))
	sb.WriteString("\n")
	sb.WriteString(sMuted.Render(" Manager: "))
	sb.WriteString(sSuccess.Render("node-1"))
	sb.WriteString("\n")
	sb.WriteString(sMuted.Render(" Workers: "))
	sb.WriteString(sSuccess.Render("4 ready"))
	sb.WriteString(" ")
	sb.WriteString(sError.Render("1 down"))
	sb.WriteString("\n")
	sb.WriteString(sMuted.Render(" Services: "))
	sb.WriteString(sSuccess.Render("7 running"))
	sb.WriteString("\n")
	cpu := fmt.Sprintf("%.0f%%", clusterCPUPct())
	mem := fmt.Sprintf("%.0f%%", clusterMemPct())
	sb.WriteString(sMuted.Render(" CPU: " + cpu + "  MEM: " + mem))
	return sb.String()
}

func clusterCPUPct() float64 {
	total := 0.0
	count := 0
	for _, n := range mockNodes {
		if n.status == "ready" {
			total += n.cpu
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func clusterMemPct() float64 {
	total := 0.0
	count := 0
	for _, n := range mockNodes {
		if n.status == "ready" {
			total += n.mem
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// Estilos do logo e cluster info
var (
	sLogo = lipgloss.NewStyle().
		Bold(true).
		Foreground(cPrimary)

	sClusterTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cAccent)

	sStatusOnline = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(cSuccess)

	sStatusCmd = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(cPrimary)

	sStatusFilter = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(cAccent)

	sStatusHelp = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(cWarning)

	sStatusDetail = lipgloss.NewStyle().
			Bold(true).
			Foreground(cWhite).
			Background(cBorder)
)
