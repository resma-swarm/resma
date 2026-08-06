package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderLogsView renderiza a tela de logs inline (como ViewDetail).
// O usuário navega entre linhas de log com j/k, Enter expande a mensagem.
func renderLogsView(m model, height int) string {
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

	total := len(logs)
	if total == 0 {
		return sMuted.Render(" No logs found for " + m.selectedItem)
	}

	// Auto-scroll para o fim se follow estiver ON
	cursor := m.logCursor
	if m.logFollow {
		cursor = total - 1
	}

	// Calcular viewport
	// -2 linhas: título + separador inferior
	// -1 linha: status bar
	visible := height - 3
	if visible < 3 {
		visible = 3
	}

	// Calcular janela visível centrada no cursor
	startIdx := cursor - visible/2
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visible
	if endIdx > total {
		endIdx = total
		startIdx = endIdx - visible
		if startIdx < 0 {
			startIdx = 0
		}
	}

	var sb strings.Builder

	// Título do painel
	title := sClusterTitle.Render(" LOGS: " + m.selectedItem + " ")
	filterInfo := ""
	if m.logFilter != "" {
		filterInfo = sHighlight.Render(" filter:" + m.logFilter + " ")
	}
	followInfo := ""
	if m.logFollow {
		followInfo = sSuccess.Render(" FOLLOW ")
	} else {
		followInfo = sMuted.Render(" PAUSED ")
	}
	countInfo := sMuted.Render(" " + itoa(startIdx+1) + "-" + itoa(endIdx) + "/" + itoa(total) + " ")

	titleLine := lipgloss.JoinHorizontal(lipgloss.Left, title, filterInfo, followInfo, countInfo)
	sb.WriteString(lipgloss.NewStyle().Width(m.width - 2).Render(titleLine))
	sb.WriteString("\n")

	// Renderizar linhas de log
	for i := startIdx; i < endIdx; i++ {
		l := logs[i]
		isSelected := i == cursor
		sb.WriteString(renderLogLine(l, m.width-2, isSelected))
		sb.WriteString("\n")
	}

	// Preencher linhas vazias
	renderedLines := endIdx - startIdx
	for i := renderedLines; i < visible; i++ {
		sb.WriteString("\n")
	}

	// Status bar
	statusItems := []string{
		"[j/k] navigate",
		"[Enter] expand",
		"[f] follow",
		"[/] filter",
		"[Esc] back",
	}
	statusBar := sMuted.Render(strings.Join(statusItems, "  "))
	sb.WriteString(lipgloss.NewStyle().Width(m.width - 2).Render(statusBar))

	return sb.String()
}

// renderLogLine renderiza uma linha de log com cores por nível.
// Se selected, aplica background highlight (cursor).
func renderLogLine(l mockLogEntry, width int, selected bool) string {
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

	// Largura do prefixo sem ANSI: timestamp(20) + space + level(5) + "  " = 28
	prefixW := len(l.timestamp) + 1 + 5 + 2
	msgW := width - prefixW
	if msgW < 10 {
		msgW = 10
	}

	msg := l.message
	if len(msg) > msgW {
		msg = msg[:msgW-1] + "…"
	}

	line := tsStyled + " " + levelStyled + "  " + msg

	if selected {
		return sTableCursor.Width(width).Render(line)
	}
	return lipgloss.NewStyle().Width(width).Render(line)
}

// renderLogPopup renderiza um popup centralizado com a mensagem completa do log.
func renderLogPopup(m model) string {
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

	if m.logCursor < 0 || m.logCursor >= len(logs) {
		return ""
	}
	l := logs[m.logCursor]

	// Largura do popup: 80% da tela, max 100
	popupW := m.width * 80 / 100
	if popupW > 100 {
		popupW = 100
	}
	if popupW < 40 {
		popupW = 40
	}

	// Quebrar mensagem em linhas de popupW-4 chars
	msgLines := wrapText(l.message, popupW-4)

	// Altura do popup: header(3) + msgLines + footer(1)
	popupH := 3 + len(msgLines) + 1
	if popupH > m.height-4 {
		popupH = m.height - 4
	}

	// Construir conteúdo do popup
	var content strings.Builder
	content.WriteString(sClusterTitle.Render(" LOG ENTRY DETAIL "))
	content.WriteString("\n")
	content.WriteString(sMuted.Render(strings.Repeat("─", popupW-2)))
	content.WriteString("\n")

	var levelStyled string
	switch l.level {
	case "ERROR":
		levelStyled = sError.Render("ERROR")
	case "WARN":
		levelStyled = sWarning.Render("WARN")
	case "DEBUG":
		levelStyled = sMuted.Render("DEBUG")
	default:
		levelStyled = sSuccess.Render("INFO")
	}

	content.WriteString(sInfoKey.Render("Time:  ") + sInfoVal.Render(l.timestamp) + "  " + levelStyled)
	content.WriteString("\n")
	content.WriteString(sMuted.Render(strings.Repeat("─", popupW-2)))
	content.WriteString("\n")

	for _, ml := range msgLines {
		content.WriteString(ml)
		content.WriteString("\n")
	}

	popupContent := content.String()

	// Estilo do popup com borda
	popup := lipgloss.NewStyle().
		Width(popupW).
		Height(popupH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cResmaPrimary).
		Background(cResmaTabBg).
		Padding(0, 1).
		Render(popupContent)

	// Centralizar na tela
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, popup)
}

// wrapText quebra texto em linhas de no máximo width chars.
func wrapText(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) <= width {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	lines = append(lines, current)
	return lines
}
