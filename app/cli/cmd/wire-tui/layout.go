package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth  = 80
	minHeight = 24
)

// renderDashboard é o entry point do View().
func renderDashboard(m model) string {
	if m.quitting {
		return ""
	}

	// Splash screen no startup
	if m.splash {
		return renderSplash(m)
	}

	// Terminal muito pequeno
	if m.width < minWidth || m.height < minHeight {
		return sError.Render(
			"Terminal too small: " + itoa(m.width) + "x" + itoa(m.height) +
				" (min: " + itoa(minWidth) + "x" + itoa(minHeight) + ")\nPress q to quit")
	}

	// Help overlay (fullscreen)
	if m.viewMode == ViewHelp {
		return renderHelp(m)
	}

	// Layout principal estilo k9s:
	//   Header (clusterInfo + menu + logo)  — 7 linhas
	//   Content (two-column ou detail)      — flex
	//   Crumbs                               — 1 linha
	//   Flash                                — 1 linha
	//   Prompt (se command/filter ativo)     — 3 linhas

	header := renderHeaderRich(m)
	content := renderContent(m)
	crumbs := renderCrumbs(m)
	flash := renderFlash(m)

	parts := []string{header, content}
	if crumbs != "" {
		parts = append(parts, crumbs)
	}
	if flash != "" {
		parts = append(parts, flash)
	}

	prompt := renderPrompt(m)
	if prompt != "" {
		parts = append(parts, prompt)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderSplash mostra o logo por 1 tick no startup.
func renderSplash(m model) string {
	logo := renderLogo(m)
	tagline := sMuted.Render("  Docker Swarm Resource Manager")
	version := sMuted.Render("  v0.1.0-wireframe")
	return lipgloss.JoinVertical(lipgloss.Center,
		"\n\n\n",
		logo,
		tagline,
		version,
		"\n\n",
		sMuted.Render("  Loading..."),
	)
}

// renderHeaderRich renderiza o header com 3 seções (k9s style).
func renderHeaderRich(m model) string {
	clusterInfo := renderClusterInfo(m)
	menu := renderMenu(m)
	logo := renderLogo(m)

	// Larguras: clusterInfo ~25, logo ~26, menu = resto
	availW := m.width
	logoW := 26
	clusterW := 25
	menuW := availW - logoW - clusterW
	if menuW < 30 {
		menuW = 30
	}

	clusterStyled := lipgloss.NewStyle().Width(clusterW).Render(clusterInfo)
	menuStyled := lipgloss.NewStyle().Width(menuW).Render(menu)
	logoStyled := lipgloss.NewStyle().Width(logoW).Render(logo)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		clusterStyled, menuStyled, logoStyled)
}

// renderContent roteia para o modo de visualização atual.
func renderContent(m model) string {
	switch m.viewMode {
	case ViewDetail:
		return renderDetailView(m)
	default:
		return renderListView(m)
	}
}

// renderListView renderiza o layout two-column (side + main).
func renderListView(m model) string {
	side := renderSidePanel(m)
	main := renderMainPanel(m)

	// Bordas indicam foco
	if m.focusedPanel == PanelSide {
		side = sBorderActive.Render(side)
		main = sBorderInactive.Render(main)
	} else {
		side = sBorderInactive.Render(side)
		main = sBorderActive.Render(main)
	}

	// Calcular larguras disponíveis
	availW := m.width - 3 // borders + padding
	sideW := availW / 3
	mainW := availW - sideW

	// Garantir que cada painel tenha a largura correta
	side = lipgloss.NewStyle().Width(sideW - 2).Render(side)
	main = lipgloss.NewStyle().Width(mainW - 2).Render(main)

	return lipgloss.JoinHorizontal(lipgloss.Top, side, main)
}

// renderSidePanel renderiza a lista lateral filtrável.
func renderSidePanel(m model) string {
	items := m.currentItems()
	var sb strings.Builder
	sb.WriteString(sMuted.Render(" Filter: " + m.filter + "\n"))
	sb.WriteString(sMuted.Render(strings.Repeat("─", 20)))
	sb.WriteString("\n")
	for i, item := range items {
		marker := "  "
		if i == m.cursor {
			marker = "▶ "
		}
		if i == m.cursor {
			sb.WriteString(sSelected.Render(marker + truncate(item, 18)))
		} else {
			sb.WriteString(marker + truncate(item, 18))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderMainPanel roteia para a tab ativa.
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

// currentItems retorna os nomes dos itens da tab ativa.
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

// listLen retorna o número de itens da tab ativa.
func (m model) listLen() int {
	return len(m.currentItems())
}

// itoa helper simples.
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
