package tui

import (
	"strings"
)

// renderTabBar renderiza a régua visual de abas com suporte aos números 1-6 e Tab
func renderTabBar(m model) string {
	var sb strings.Builder
	for i, name := range tabNames {
		if TabID(i) == m.activeTab {
			sb.WriteString(sTabActive.Render(name))
		} else {
			sb.WriteString(sTabInactive.Render(name))
		}
	}
	return sb.String()
}
