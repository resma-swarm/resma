package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth  = 80
	minHeight = 24
)

func renderDashboard(m model) string {
	if m.quitting {
		return ""
	}
	if m.splash {
		return renderSplash(m)
	}

	if m.width < minWidth || m.height < minHeight {
		return sError.Render("Terminal too small!")
	}

	if m.viewMode == ViewHelp {
		return renderHelp(m)
	}

	// 1. Header Superior Rico
	header := renderHeaderRich(m)
	headerLines := strings.Count(header, "\n") + 1

	// 2. Barra Visual de Abas (1 linha)
	tabs := renderTabBar(m)
	tabLines := strings.Count(tabs, "\n") + 1

	// 3. Crumbs, Flash, Prompt — medir linhas reais de cada um
	crumbs := renderCrumbs(m)
	flash := renderFlash(m)
	prompt := renderPrompt(m)

	crumbsLines := 0
	if crumbs != "" {
		crumbsLines = strings.Count(crumbs, "\n") + 1
	}
	flashLines := 0
	if flash != "" {
		flashLines = strings.Count(flash, "\n") + 1
	}
	promptLines := 0
	if prompt != "" {
		promptLines = strings.Count(prompt, "\n") + 1
	}

	// 4. Content area: altura = terminal - header - tabs - crumbs - flash - prompt
	// O content area tem borda top+bottom (2 linhas) que lipgloss adiciona
	// AFTER Height(), então precisamos passar contentHeight - 2 para Height()
	// para que o total (com bordas) seja contentHeight.
	contentHeight := m.height - headerLines - tabLines - crumbsLines - flashLines - promptLines
	if contentHeight < 5 {
		contentHeight = 5
	}

	content := renderContentArea(m, contentHeight)

	// 5. Montar dashboard
	parts := []string{header, tabs, content}
	if crumbs != "" {
		parts = append(parts, crumbs)
	}
	if flash != "" {
		parts = append(parts, flash)
	}
	if prompt != "" {
		parts = append(parts, prompt)
	}

	dashboard := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// 6. SAFETY NET: garantir que o dashboard tem EXATAMENTE m.height linhas
	// e cada linha tem no máximo m.width chars visuais.
	// O lipgloss.Width() em algumas versões não trunca, apenas wrapa/padeia,
	// o que causa overflow visual no terminal.
	lines := strings.Split(dashboard, "\n")

	// Truncar cada linha para m.width chars visuais
	for i, line := range lines {
		if visualWidth(line) > m.width {
			lines[i] = truncateAnsi(line, m.width)
		}
	}

	// Garantir exatamente m.height linhas
	if len(lines) > m.height {
		lines = lines[:m.height]
	} else if len(lines) < m.height {
		for i := len(lines); i < m.height; i++ {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

func renderContentArea(m model, height int) string {
	var body string
	switch m.viewMode {
	case ViewDetail:
		body = renderDetailView(m)
	case ViewLogs:
		body = renderLogsView(m, height)
	case ViewLogDetail:
		body = renderLogDetailView(m, height)
	default:
		body = renderMainPanel(m)
	}

	// lipgloss aplica Height() ANTES da borda, então a borda adiciona
	// 2 linhas extras (top + bottom). Passar height-2 para Height()
	// faz com que o total (conteúdo + bordas) seja exatamente height.
	innerH := height - 2
	if innerH < 3 {
		innerH = 3
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(innerH).
		Border(lipgloss.NormalBorder(), true, false, true, false).
		BorderForeground(cResmaBorder).
		Render(body)
}

func renderMainPanel(m model) string {
	switch m.activeTab {
	case TabServices:
		return renderServicesTab(m)
	case TabNodes:
		return renderNodesTab(m)
	case TabAgents:
		return renderAgentsTab(m)
	case TabTasks:
		return renderTasksTab(m)
	case TabAlerts:
		return renderAlertsTab(m)
	case TabRecommendations:
		return renderRecommendationsTab(m)
	}
	return ""
}

func renderSplash(m model) string {
	logo := "  ___ ___ ___ __  __   _   \n" +
		" | _ \\ __/ __|  \\/  | /_\\  \n" +
		" |   / _|\\__ \\ |\\/| |/ _ \\ \n" +
		" |_|_\\___|___/_|  |_/_/ \\_\\"
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			sLogo.Render(logo),
			"\n",
			sInfoVal.Render("Docker Swarm Resource Manager"),
			sMuted.Render("v0.1.0-wireframe"),
			"\n",
			sMuted.Render("Iniciando Dashboard..."),
		))
}

func (m model) currentItems() []string {
	switch m.activeTab {
	case TabServices:
		var r []string
		for _, s := range mockServices {
			r = append(r, s.name)
		}
		return r
	case TabNodes:
		var r []string
		for _, n := range mockNodes {
			r = append(r, n.id)
		}
		return r
	case TabAgents:
		var r []string
		for _, a := range mockAgents {
			r = append(r, a.nodeID)
		}
		return r
	case TabTasks:
		var r []string
		for _, t := range mockTasks {
			r = append(r, t.id)
		}
		return r
	case TabAlerts:
		var r []string
		for _, a := range mockAlerts {
			r = append(r, a.service)
		}
		return r
	case TabRecommendations:
		var r []string
		for _, rc := range mockRecs {
			r = append(r, rc.service)
		}
		return r
	}
	return nil
}

func (m model) listLen() int {
	return len(m.currentItems())
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
