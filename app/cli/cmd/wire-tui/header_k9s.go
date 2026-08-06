package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHeaderRich renderiza o header 1:1 com k9s.
func renderHeaderRich(m model) string {
	// 1. ClusterInfo (Esquerda)
	clusterInfo := renderK9sClusterInfo(m)
	
	// 2. Menu Grid (Centro)
	menuGrid := renderK9sMenu(m)
	
	// 3. Logo & Status (Direita)
	logoSection := renderK9sLogo(m)

	// Calcular larguras
	logoW := 26
	infoW := 35
	menuW := m.width - logoW - infoW
	if menuW < 0 { menuW = 0 }

	infoStyled := lipgloss.NewStyle().Width(infoW).Render(clusterInfo)
	menuStyled := lipgloss.NewStyle().Width(menuW).Render(menuGrid)
	logoStyled := lipgloss.NewStyle().Width(logoW).Render(logoSection)

	return lipgloss.JoinHorizontal(lipgloss.Top, infoStyled, menuStyled, logoStyled)
}

func renderK9sClusterInfo(m model) string {
	var sb strings.Builder
	sb.WriteString(sK9sClusterTitle.Render(" Context: default "))
	sb.WriteString("\n")
	sb.WriteString(sK9sInfoKey.Render(" Cluster: ") + sK9sInfoVal.Render("resma-swarm"))
	sb.WriteString("\n")
	sb.WriteString(sK9sInfoKey.Render(" User:    ") + sK9sInfoVal.Render("admin"))
	sb.WriteString("\n")
	sb.WriteString(sK9sInfoKey.Render(" K9s:     ") + sK9sInfoVal.Render("v0.32.4"))
	sb.WriteString("\n")
	sb.WriteString(sK9sInfoKey.Render(" CPU:     ") + renderK9sBar(65, cK9sGreen))
	sb.WriteString("\n")
	sb.WriteString(sK9sInfoKey.Render(" MEM:     ") + renderK9sBar(42, cK9sGreen))
	return sb.String()
}

func renderK9sBar(pct int, color lipgloss.Color) string {
	width := 15
	filled := (pct * width) / 100
	bar := strings.Repeat("■", filled) + strings.Repeat("□", width-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar) + fmt.Sprintf(" %d%%", pct)
}

func renderK9sLogo(m model) string {
	logo := " ____  ____  ____  _  _ \n" +
		"|  _ \\|  _ \\/ ___|| || |\n" +
		"| |_) | |_) \\___ \\| || |\n" +
		"|  _ <|  __/ ___) |__   |\n" +
		"|_| \\_\\_|  |____/   |_/ |"

	status := " ONLINE "
	style := sK9sStatus.Copy().Background(cK9sGreen)
	
	switch m.viewMode {
	case ViewCommand:
		status = " COMMAND "
		style = sK9sStatus.Copy().Background(cK9sPrimary)
	case ViewFilter:
		status = " FILTER "
		style = sK9sStatus.Copy().Background(cK9sAqua)
	case ViewHelp:
		status = " HELP "
		style = sK9sStatus.Copy().Background(cK9sWarning)
	}

	return lipgloss.JoinVertical(lipgloss.Center,
		sK9sLogo.Render(logo),
		"\n",
		style.Render(status),
	)
}

func renderK9sMenu(m model) string {
	hints := menuHints(m)
	const maxRows = 6
	var rows [maxRows][]string

	for i, h := range hints {
		row := i % maxRows
		key := sK9sMenuKey.Render("<" + h.Key + ">")
		desc := sK9sMenuDesc.Render(h.Desc)
		rows[row] = append(rows[row], fmt.Sprintf(" %s %s", key, desc))
	}

	var sb strings.Builder
	for i := 0; i < maxRows; i++ {
		sb.WriteString(strings.Join(rows[i], "  "))
		sb.WriteString("\n")
	}
	return sb.String()
}
