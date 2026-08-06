package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderLogsView renderiza a tela de logs estilo k9s — fullscreen, com
// header do serviço, filtros, timestamps coloridos por nível, e indicador
// de follow/scroll.
func renderLogsView(m model) string {
	logs := mockLogsFor(m.selectedItem)

	// Aplicar filtro se houver
	if m.logFilter != "" {
		filtered := make([]mockLogEntry, 0)
		for _, l := range logs {
			if strings.Contains(strings.ToLower(l.message), strings.ToLower(m.logFilter)) ||
				strings.Contains(strings.ToLower(l.level), strings.ToLower(m.logFilter)) {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	// Calcular área útil
	reservedHeight := 11 // header + tabs + crumbs + flash
	contentHeight := m.height - reservedHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Auto-scroll para o fim se follow estiver ON
	if m.logFollow && len(logs) > 0 {
		m.logScroll = len(logs) - 1
	}

	// Calcular janela visível (viewport)
	total := len(logs)
	visible := contentHeight - 3 // -3 para título + status bar
	if visible < 1 {
		visible = 1
	}

	// Determinar range de logs visíveis
	startIdx := 0
	endIdx := total
	if total > visible {
		// Se follow, mostrar os últimos
		if m.logFollow {
			startIdx = total - visible
			endIdx = total
		} else {
			// Scroll manual
			startIdx = m.logScroll
			if startIdx > total-visible {
				startIdx = total - visible
			}
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx = startIdx + visible
			if endIdx > total {
				endIdx = total
			}
		}
	}

	var sb strings.Builder

	// Título do painel de logs
	title := sClusterTitle.Render(" LOGS: " + m.selectedItem + " ")
	filterInfo := ""
	if m.logFilter != "" {
		filterInfo = sHighlight.Render(" filter: "+m.logFilter+" ")
	}
	followInfo := ""
	if m.logFollow {
		followInfo = sSuccess.Render(" [FOLLOW] ")
	} else {
		followInfo = sMuted.Render(" [PAUSED] ")
	}
	countInfo := sMuted.Render(" " + itoa(startIdx+1) + "-" + itoa(endIdx) + "/" + itoa(total) + " lines ")

	// Montar linha de título
	titleLine := lipgloss.JoinHorizontal(lipgloss.Left,
		title, filterInfo, followInfo, countInfo)
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(titleLine))
	sb.WriteString("\n")

	// Linha separadora
	sb.WriteString(sMuted.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")

	// Renderizar linhas de log
	for i := startIdx; i < endIdx && i < total; i++ {
		l := logs[i]
		sb.WriteString(renderLogLine(l, m.width))
		sb.WriteString("\n")
	}

	// Preencher linhas vazias se sobrar espaço
	renderedLines := endIdx - startIdx
	for i := renderedLines; i < visible; i++ {
		sb.WriteString("\n")
	}

	// Status bar inferior
	sb.WriteString(sMuted.Render(strings.Repeat("─", m.width)))
	sb.WriteString("\n")
	statusItems := []string{
		"[j/k] scroll",
		"[f] follow",
		"[/] filter",
		"[g/G] top/bottom",
		"[Esc] back",
	}
	statusBar := sMuted.Render(strings.Join(statusItems, "  "))
	sb.WriteString(lipgloss.NewStyle().Width(m.width).Render(statusBar))

	return lipgloss.NewStyle().Width(m.width).Height(contentHeight).Render(sb.String())
}

// renderLogLine renderiza uma linha de log com cores por nível.
func renderLogLine(l mockLogEntry, width int) string {
	var levelStyled string
	switch l.level {
	case "ERROR":
		levelStyled = sError.Render("ERROR")
	case "WARN":
		levelStyled = sWarning.Render("WARN ")
	case "DEBUG":
		levelStyled = sMuted.Render("DEBUG")
	default:
		levelStyled = sSuccess.Render("INFO ")
	}

	tsStyled := sMuted.Render(l.timestamp)

	// Calcular largura do prefixo (timestamp + level) sem ANSI
	prefixW := len(l.timestamp) + 1 + 5 + 2 // ts + space + level + "  "
	msgW := width - prefixW
	if msgW < 10 {
		msgW = 10
	}

	msg := l.message
	if len(msg) > msgW {
		msg = msg[:msgW-1] + "…"
	}

	return tsStyled + " " + levelStyled + "  " + msg
}
