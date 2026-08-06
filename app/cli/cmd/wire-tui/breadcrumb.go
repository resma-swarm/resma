package main

import "strings"

func renderBreadcrumb(m model) string {
	tabLabel := tabNames[m.activeTab][4:] // remove "[N] "
	parts := []string{tabLabel}
	if m.viewMode == ViewDetail && m.selectedItem != "" {
		parts = append(parts, m.selectedItem)
	}
	return sBreadcrumb.Render(strings.Join(parts, " > "))
}
