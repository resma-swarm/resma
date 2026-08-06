package tui

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

	// Regras de largura — balancear as 3 seções
	logoW := 28
	infoW := 28
	menuW := m.width - logoW - infoW
	if menuW < 0 {
		menuW = 0
	}

	infoStyled := lipgloss.NewStyle().Width(infoW).Render(clusterInfo)
	menuStyled := lipgloss.NewStyle().Width(menuW).Render(menuGrid)
	logoStyled := lipgloss.NewStyle().Width(logoW).Render(logoSection)

	header := lipgloss.JoinHorizontal(lipgloss.Top, infoStyled, menuStyled, logoStyled)

	// Garantir que cada linha do header tem exatamente m.width chars visuais.
	// Se o menu for mais largo que menuW (Width não trunca, apenas wrap/pad),
	// truncar cada linha para evitar que o terminal quebre linhas largas.
	headerLines := strings.Split(header, "\n")
	for i, line := range headerLines {
		if visualWidth(line) > m.width {
			headerLines[i] = truncateAnsi(line, m.width)
		}
	}
	return strings.Join(headerLines, "\n")
}

func renderClusterInfo(m model) string {
	var sb strings.Builder
	sb.WriteString(sClusterTitle.Render(" Context: default "))
	sb.WriteString("\n")

	// Stack e Role — usar dados reais se disponíveis, fallback mock
	stackName := "resma-swarm"
	role := "manager"
	nodesReady := "1/1"
	if m.cluster != nil {
		if m.cluster.Cluster.ID != "" {
			role = m.cluster.NodesDistribution[0].Role
		}
		nodesReady = fmt.Sprintf("%d/%d", m.cluster.Cluster.NodesReady, m.cluster.Cluster.NodesTotal)
	}

	sb.WriteString(sInfoKey.Render(" Stack: ") + sInfoVal.Render(stackName))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" Role:  ") + sInfoVal.Render(role))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" Nodes: ") + sInfoVal.Render(nodesReady))
	sb.WriteString("\n")

	// CPU e MEM — dados reais do cluster via SSE
	cpuPct, memPct, memUsed, memTotal := clusterMetrics(m)
	cpuColor := metricColor(cpuPct)
	memColor := metricColor(memPct)

	// Efeito flash: se o valor acabou de mudar via SSE, renderizar em bold white
	cpuBar := renderMetricBar(cpuPct, cpuColor)
	if m.cpuFlashing() {
		cpuBar = renderMetricBarFlash(cpuPct)
	}
	memBar := renderMetricBar(memPct, memColor)
	memSuffix := fmt.Sprintf(" %.1fG/%.1fG", memUsed, memTotal)
	if m.memFlashing() {
		memBar = renderMetricBarFlash(memPct)
		memSuffix = sFlash.Render(memSuffix)
	}

	sb.WriteString(sInfoKey.Render(" CPU:   ") + cpuBar)
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render(" MEM:   ") + memBar + memSuffix)
	return sb.String()
}

// clusterMetrics extrai CPU% e MEM% do payload SSE, com fallback mock.
// Retorna: cpuPct (0-100), memPct (0-100), memUsedGB, memTotalGB.
func clusterMetrics(m model) (cpuPct, memPct, memUsedGB, memTotalGB float64) {
	if m.cluster == nil {
		// Fallback mock quando não há dados SSE ainda
		return 65, 42, 3.4, 8.0
	}
	cap := &m.cluster.ClusterCapacity
	cpuPct = cap.CPUPercent()
	memPct = cap.MemPercent()
	memUsedGB = cap.MemUsageGB()
	memTotalGB = cap.MemTotalGB()
	return
}

// renderMetricBarFlash renderiza a barra em bold branco (efeito "valor atualizado").
// Usado quando o valor acabou de mudar via SSE — destaca por ~1.5s e volta ao normal.
func renderMetricBarFlash(pct float64) string {
	width := 10
	filled := int((pct * float64(width)) / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("■", filled) + strings.Repeat("□", width-filled)
	return sFlash.Render(bar) + sFlash.Render(fmt.Sprintf(" %3.0f%%", pct))
}

// metricColor retorna a cor adequada para uma porcentagem de uso.
//   - < 70%: verde
//   - 70-89%: amarelo (warning)
//   - >= 90%: vermelho (danger)
func metricColor(pct float64) lipgloss.Color {
	if pct >= 90 {
		return cResmaRed
	}
	if pct >= 70 {
		return cResmaWarning
	}
	return cResmaGreen
}

func renderMetricBar(pct float64, color lipgloss.Color) string {
	width := 10
	filled := int((pct * float64(width)) / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("■", filled) + strings.Repeat("□", width-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar) + fmt.Sprintf(" %3.0f%%", pct)
}

// renderBrandSection renderiza o logo ASCII + status bar.
// O logo é renderizado como um bloco único (sem JoinVertical com separadores)
// para preservar todos os caracteres incluindo underscores do topo.
func renderBrandSection(m model) string {
	// Figlet "Small" — 4 linhas de logo + 1 linha em branco no topo
	// para alinhar com a ClusterInfo (6 linhas).
	logoBlock := "                          \n" +
		" ___ ___ ___ __  __   _   \n" +
		"| _ \\ __/ __|  \\/  | /_\\  \n" +
		"|   / _|\\__ \\ |\\/| |/ _ \\ \n" +
		"|_|_\\___|___/_|  |_/_/ \\_\\\n" +
		"                          \n"

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
		Width(26).
		AlignHorizontal(lipgloss.Center).
		Render(style.Render(status))

	return logoStyled + statusStyled
}

func renderMenu(m model) string {
	hints := menuHints(m)
	const maxRows = 5

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
