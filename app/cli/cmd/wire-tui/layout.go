package main

import (
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

	// 1. Header Superior Rico (8 linhas)
	header := renderHeaderRich(m)

	// 2. Barra Visual de Abas
	tabs := renderTabBar(m)

	// 3. Conteúdo Principal (Ajuste para dar espaço com respiro)
	reservedHeight := 11
	if m.viewMode == ViewCommand || m.viewMode == ViewFilter {
		reservedHeight += 3
	}
	contentHeight := m.height - reservedHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	content := renderContentArea(m, contentHeight)

	// 4. Crumbs de Navegação
	crumbs := renderCrumbs(m)

	// 5. Mensagens Flash
	flash := renderFlash(m)

	// 6. Prompt de Entrada
	prompt := renderPrompt(m)

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

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func renderContentArea(m model, height int) string {
	var body string
	if m.viewMode == ViewDetail {
		body = renderDetailView(m)
	} else {
		body = renderMainPanel(m)
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			sLogo.Render("RESMA MONITOR"),
			sInfoVal.Render("Iniciando Dashboard..."),
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
