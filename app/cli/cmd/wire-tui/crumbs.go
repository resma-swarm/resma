package main

import (
	"fmt"
	"strings"
)

// renderCrumbs renderiza breadcrumbs no estilo RESMA — chips coloridos.
func renderCrumbs(m model) string {
	var crumbs []string

	// Tab atual como primeiro crumb
	crumbs = append(crumbs, tabNames[m.activeTab][4:]) // remove "[N] "

	// Item selecionado em drill-down
	if m.viewMode == ViewDetail && m.selectedItem != "" {
		crumbs = append(crumbs, m.selectedItem)
	}

	// Modo especial
	switch m.viewMode {
	case ViewCommand:
		crumbs = append(crumbs, "command")
	case ViewFilter:
		crumbs = append(crumbs, "filter")
	case ViewHelp:
		crumbs = append(crumbs, "help")
	}

	if len(crumbs) == 0 {
		return ""
	}

	var sb strings.Builder
	last := len(crumbs) - 1
	for i, c := range crumbs {
		name := strings.ReplaceAll(strings.ToLower(c), " ", "")
		if i == last {
			// crumb ativo — cor de destaque
			sb.WriteString(sCrumbActive.Render(fmt.Sprintf(" <%s> ", name)))
		} else {
			// crumb inativo
			sb.WriteString(sCrumbInactive.Render(fmt.Sprintf(" <%s> ", name)))
		}
		sb.WriteString(" ")
	}
	return sb.String()
}
