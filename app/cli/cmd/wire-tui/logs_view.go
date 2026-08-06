package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderLogsView renderiza a tela de logs inline usando TableModel.
// O usuário navega entre linhas com j/k, Enter abre popup com mensagem completa.
func renderLogsView(m model, height int) string {
	logs := mockLogsFor(m.selectedItem)

	// Aplicar filtro
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

	// Construir colunas da tabela de logs
	cols := []TableColumn{
		{Title: "TIME", Width: 21, Align: lipgloss.Left},
		{Title: "LEVEL", Width: 7, Align: lipgloss.Left},
		{Title: "MESSAGE", Width: 0, Align: lipgloss.Left, Flex: true},
	}

	// Construir linhas
	rows := make([]TableRow, total)
	for i, l := range logs {
		// Level em CAIXA ALTA com cor
		var levelColored string
		var levelPlain string
		switch l.level {
		case "ERROR":
			levelColored = sError.Render("ERROR")
			levelPlain = "ERROR"
		case "WARN":
			levelColored = sWarning.Render("WARN")
			levelPlain = "WARN"
		case "DEBUG":
			levelColored = sMuted.Render("DEBUG")
			levelPlain = "DEBUG"
		default:
			levelColored = sSuccess.Render("INFO")
			levelPlain = "INFO"
		}

		// Timestamp
		tsColored := sMuted.Render(l.timestamp)
		tsPlain := l.timestamp

		// Mensagem truncada (será truncada pelo Width do cell)
		msgColored := l.message
		msgPlain := l.message

		rows[i] = TableRow{
			Cells: []string{tsColored, levelColored, msgColored},
			Plain: []string{tsPlain, levelPlain, msgPlain},
		}
	}

	// Título com info de follow/filter
	var titleParts []string
	titleParts = append(titleParts, sClusterTitle.Render(" LOGS: "+m.selectedItem+" "))
	if m.logFilter != "" {
		titleParts = append(titleParts, sHighlight.Render(" filter:"+m.logFilter+" "))
	}
	if m.logFollow {
		titleParts = append(titleParts, sSuccess.Render(" FOLLOW "))
	} else {
		titleParts = append(titleParts, sMuted.Render(" PAUSED "))
	}
	titleParts = append(titleParts, sMuted.Render(" "+itoa(m.logCursor+1)+"/"+itoa(total)+" "))
	title := strings.Join(titleParts, "")

	// Criar e configurar tabela
	table := NewTable(cols)
	table.SetWidth(m.width - 2) // -2 pela borda do content area
	table.SetHeight(height)
	table.SetHeader(title)
	table.SetRows(rows)

	// Sincronizar cursor
	// (TableModel tem seu próprio cursor, mas usamos m.logCursor)
	// Como não podemos setar cursor diretamente, copiamos o estado
	if m.logFollow {
		table.cursor = total - 1
	} else {
		table.cursor = m.logCursor
	}

	return table.View()
}

// renderLogPopupOverlay renderiza o popup sobre o dashboard completo.
// O dashboard é renderizado normalmente, e as linhas do popup são
// sobrepostas manualmente sobre as linhas do dashboard — o fundo
// permanece visível onde o popup não cobre.
func renderLogPopupOverlay(m model, dashboard string) string {
	logs := mockLogsFor(m.selectedItem)

	// Aplicar filtro
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

	cursor := m.logCursor
	if m.logFollow {
		cursor = len(logs) - 1
	}
	if cursor < 0 || cursor >= len(logs) {
		return dashboard
	}
	l := logs[cursor]

	// Largura do popup
	popupW := m.width * 80 / 100
	if popupW > 100 {
		popupW = 100
	}
	if popupW < 40 {
		popupW = 40
	}

	// Construir conteúdo do popup
	var content strings.Builder

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
	content.WriteString(sMuted.Render(strings.Repeat("─", popupW-6)))
	content.WriteString("\n")

	// Mensagem completa com word-wrap
	msgLines := wrapText(l.message, popupW-6)
	for _, ml := range msgLines {
		content.WriteString(sInfoVal.Render(ml))
		content.WriteString("\n")
	}

	// Altura do popup baseada no conteúdo
	popupH := 4 + len(msgLines) + 1 // título + separador1 + time + separador2 + msg + padding
	if popupH > m.height-4 {
		popupH = m.height - 4
	}

	// Criar popup — SEM background color (overlay)
	popupStyled := lipgloss.NewStyle().
		Width(popupW).
		Height(popupH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cResmaPrimary).
		Padding(0, 1).
		Render(sClusterTitle.Render(" LOG ENTRY DETAIL ") + "\n" +
			sMuted.Render(strings.Repeat("─", popupW-4)) + "\n" + content.String())

	// Compor overlay: sobrepor linhas do popup sobre linhas do dashboard
	return overlayText(dashboard, popupStyled, m.width, m.height, popupW, popupH)
}

// overlayText sobrepõe popupLines sobre bgLines, centralizado.
// Onde o popup tem conteúdo, sobrescreve o background.
// Onde o popup tem espaços (fora da borda), o background permanece visível.
func overlayText(bg, popup string, screenW, screenH, popupW, popupH int) string {
	bgLines := strings.Split(bg, "\n")
	popupLines := strings.Split(popup, "\n")

	// Calcular posição central
	startY := (screenH - popupH) / 2
	startX := (screenW - popupW) / 2
	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Garantir que bgLines tem pelo menos screenH linhas
	for len(bgLines) < screenH {
		bgLines = append(bgLines, strings.Repeat(" ", screenW))
	}

	// Sobrepor cada linha do popup sobre a linha correspondente do bg
	for py := 0; py < len(popupLines) && py+startY < len(bgLines); py++ {
		bgLine := bgLines[py+startY]
		popupLine := popupLines[py]

		// Garantir que bgLine tem pelo menos screenW chars
		bgRunes := []rune(bgLine)
		for len(bgRunes) < screenW {
			bgRunes = append(bgRunes, ' ')
		}

		// Sobrepor: a partir de startX, copiar chars do popup
		// Mas precisamos preservar ANSI codes do popupLine inteiro
		// Solução: pegar o prefixo do bg até startX, depois o popupLine, depois o resto do bg
		prefix := string(bgRunes[:startX])
		// Sufixo: chars do bg após startX + len(popupLine)
		popupVisualLen := lipgloss.Width(popupLine)
		suffixStart := startX + popupVisualLen
		var suffix string
		if suffixStart < len(bgRunes) {
			suffix = string(bgRunes[suffixStart:])
		}

		bgLines[py+startY] = prefix + popupLine + suffix
	}

	return strings.Join(bgLines, "\n")
}
