package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHeaderRich renderiza o cabeçalho completo do Dashboard RESMA
func renderHeaderRich(m model) string {
	// 1. Info da Stack RESMA / Swarm (Esquerda)
	clusterInfo := renderClusterInfo(m)

	// 2. Grid de Ateliê de Teclas e Atalhos (Centro)
	menuGrid := renderMenu(m)

	// 3. Brand & Status (Direita)
	logoSection := renderBrandSection(m)

	// Regras de largura — logo ASCII precisa de 30 chars mínimo
	logoW := 30
	infoW := 31
	menuW := m.width - logoW - infoW
	if menuW < 0 {
		menuW = 0
	}

	infoStyled := lipgloss.NewStyle().Width(infoW).Render(clusterInfo)
	menuStyled := lipgloss.NewStyle().Width(menuW).Render(menuGrid)
	logoStyled := lipgloss.NewStyle().Width(logoW).Render(logoSection)

	return lipgloss.JoinHorizontal(lipgloss.Top, infoStyled, menuStyled, logoStyled)
}

func renderClusterInfo(m model) string {
	var sb strings.Builder
	sb.WriteString(sClusterTitle.Render(" Context: default "))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" Stack:   ") + sInfoVal.Render("resma-swarm"))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" Role:    ") + sInfoVal.Render("manager"))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" CLI:     ") + sInfoVal.Render("v0.1.0-wireframe"))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" CPU:     ") + renderMetricBar(65, cResmaGreen))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" MEM:     ") + renderMetricBar(42, cResmaGreen))
	return sb.String()
}

func renderMetricBar(pct int, color lipgloss.Color) string {
	width := 12
	filled := (pct * width) / 100
	bar := strings.Repeat("■", filled) + strings.Repeat("□", width-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar) + fmt.Sprintf(" %d%%", pct)
}

// renderBrandSection renderiza o logo ASCII + status bar.
// O logo é renderizado como um bloco único (sem JoinVertical com separadores)
// para preservar todos os caracteres incluindo underscores do topo.
func renderBrandSection(m model) string {
	// Figlet "Small" — 4 linhas de logo + 1 linha em branco no topo
	// para alinhar com a ClusterInfo (6 linhas).
	// Todas as linhas têm exatamente 28 chars para consistência.
	logoBlock := "                            \n" +
		"  ___ ___ ___ __  __   _    \n" +
		" | _ \\ __/ __|  \\/  | /_\\   \n" +
		" |   / _|\\__ \\ |\\/| |/ _ \\  \n" +
		" |_|_\\___|___/_|  |_/_/ \\_\\ \n" +
		"                            \n"

	status := " ONLINE "
	style := sStatus.Copy().Background(cResmaGreen)

	switch m.viewMode {
	case ViewCommand:
		status = " COMMAND "
		style = sStatus.Copy().Background(cResmaPrimary)
	case ViewFilter:
		status = " FILTER "
		style = sStatus.Copy().Background(cResmaAqua)
	case ViewHelp:
		status = " HELP "
		style = sStatus.Copy().Background(cResmaWarning)
	}

	// Logo colorido como bloco único
	logoStyled := sLogo.Render(logoBlock)

	// Status centralizado na largura do logo
	statusStyled := lipgloss.NewStyle().
		Width(28).
		AlignHorizontal(lipgloss.Center).
		Render(style.Render(status))

	return logoStyled + statusStyled
}

func renderMenu(m model) string {
	hints := menuHints(m)
	const maxRows = 6

	// Distribuir hints em colunas de maxRows linhas cada
	colCount := (len(hints) + maxRows - 1) / maxRows
	cols := make([][]KeyHint, colCount)
	for i, h := range hints {
		c := i / maxRows
		cols[c] = append(cols[c], h)
	}

	// Calcular largura máxima de cada coluna (key+desc+espaços)
	colWidths := make([]int, colCount)
	for c, col := range cols {
		maxW := 0
		for _, h := range col {
			w := len(h.Key) + 4 + len(h.Desc)
			if w > maxW {
				maxW = w
			}
		}
		colWidths[c] = maxW
	}

	var sb strings.Builder
	for row := 0; row < maxRows; row++ {
		for c := 0; c < colCount; c++ {
			if row >= len(cols[c]) {
				sb.WriteString(strings.Repeat(" ", colWidths[c]+3))
				continue
			}
			h := cols[c][row]
			key := sMenuKey.Render("[" + h.Key + "]")
			desc := sMenuDesc.Render(h.Desc)
			visW := len(h.Key) + 4 + len(h.Desc)
			pad := colWidths[c] - visW
			if pad < 0 {
				pad = 0
			}
			sb.WriteString(fmt.Sprintf(" %s %s%s   ", key, desc, strings.Repeat(" ", pad)))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
