package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderLogsView renderiza a lista de logs inline usando TableModel.
func renderLogsView(m model, height int) string {
	logs := mockLogsFor(m.selectedItem)

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

	cols := []TableColumn{
		{Title: "TIME", Width: 21, Align: lipgloss.Left},
		{Title: "LEVEL", Width: 7, Align: lipgloss.Left},
		{Title: "MESSAGE", Width: 0, Align: lipgloss.Left, Flex: true},
	}

	rows := make([]TableRow, total)
	for i, l := range logs {
		var levelColored, levelPlain string
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

		rows[i] = TableRow{
			Cells: []string{sMuted.Render(l.timestamp), levelColored, l.message},
			Plain: []string{l.timestamp, levelPlain, l.message},
		}
	}

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
	cursor := m.logCursor
	if m.logFollow {
		cursor = total - 1
	}
	titleParts = append(titleParts, sMuted.Render(" "+itoa(cursor+1)+"/"+itoa(total)+" "))
	title := strings.Join(titleParts, "")

	table := NewTable(cols)
	table.SetWidth(m.width - 2)
	table.SetHeight(height)
	table.SetHeader(title)
	table.SetRows(rows)

	if m.logFollow {
		table.cursor = total - 1
	} else {
		table.cursor = m.logCursor
	}

	return table.View()
}

// renderLogDetailView renderiza uma linha de log em detalhe (inline, não popup).
// Usa todo o espaço do content area para exibir mensagens grandes (JSON, stack traces).
// j/k navega entre linhas da mensagem, Esc volta para a lista de logs.
func renderLogDetailView(m model, height int) string {
	logs := mockLogsFor(m.selectedItem)

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
		return sMuted.Render(" No log entry selected")
	}
	l := logs[cursor]

	var sb strings.Builder

	// Título
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

	titleLine := lipgloss.JoinHorizontal(lipgloss.Left,
		sClusterTitle.Render(" LOG DETAIL: "+m.selectedItem+" "),
		sMuted.Render(" "+itoa(cursor+1)+"/"+itoa(len(logs))+" "),
		levelStyled,
	)
	sb.WriteString(lipgloss.NewStyle().Width(m.width - 2).Render(titleLine))
	sb.WriteString("\n")

	// Metadata
	sb.WriteString(sInfoKey.Render("Time:    ") + sInfoVal.Render(l.timestamp))
	sb.WriteString("\n")
	sb.WriteString(sInfoKey.Render("Level:   ") + levelStyled)
	sb.WriteString("\n")
	sb.WriteString(sMuted.Render(strings.Repeat("─", m.width-4)))
	sb.WriteString("\n")

	// Mensagem completa com word-wrap
	contentW := m.width - 4
	if contentW < 20 {
		contentW = 20
	}
	msgLines := wrapText(l.message, contentW)
	for _, ml := range msgLines {
		sb.WriteString(sInfoVal.Render(ml))
		sb.WriteString("\n")
	}

	// Preencher linhas vazias até preencher a altura
	rendered := 5 + len(msgLines) // título + 2 meta + separador + msg
	for i := rendered; i < height-1; i++ {
		sb.WriteString("\n")
	}

	// Status bar
	sb.WriteString(sMuted.Render(strings.Repeat("─", m.width-4)))
	sb.WriteString("\n")
	statusBar := sMuted.Render("[Esc] back to logs    [j/k] navigate log entries")
	sb.WriteString(lipgloss.NewStyle().Width(m.width - 2).Render(statusBar))

	return sb.String()
}
