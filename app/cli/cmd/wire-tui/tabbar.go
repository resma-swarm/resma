package main

import "strings"

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
