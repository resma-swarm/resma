package main

import "strings"

func renderFooter(m model) string {
	var items []string
	switch m.viewMode {
	case ViewList:
		items = []string{
			"q quit", "j/k move", "Enter detail", "/ filter",
			": command", "? help",
		}
		switch m.activeTab {
		case TabRecommendations:
			items = append(items, "a apply")
		case TabServices:
			items = append(items, "a apply", "d delete", "l logs")
		case TabNodes:
			items = append(items, "d drain", "l logs")
		}
	case ViewDetail:
		items = []string{"Esc back", "j/k scroll", "a apply", "e edit", "l logs", "? help"}
	case ViewFilter:
		items = []string{"Enter apply", "Esc cancel", "? help"}
	case ViewCommand:
		items = []string{"Enter execute", "Esc cancel", "? help"}
	case ViewHelp:
		items = []string{"q/Esc close"}
	}
	return sFooter.Render(strings.Join(items, " · "))
}
