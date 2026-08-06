package tui

// KeyHint representa uma keybinding exibida no menu.
type KeyHint struct {
	Key  string // ex: "0", "a", "Enter"
	Desc string // ex: "Services", "Apply"
}

// menuHints retorna as keyhints ativas para o contexto atual.
// Tab nav (0-5) é omitida — já está na tab bar.
func menuHints(m model) []KeyHint {
	var hints []KeyHint

	// Ações por view mode
	switch m.viewMode {
	case ViewList:
		hints = append(hints,
			KeyHint{"Enter", "Detail"},
			KeyHint{"l", "Logs"},
			KeyHint{"S+←/→", "SelCol"},
			KeyHint{"S+↑/↓", "Sort"},
			KeyHint{"/", "Filter"},
			KeyHint{":", "Cmd"},
		)
		switch m.activeTab {
		case TabRecommendations:
			hints = append(hints, KeyHint{"a", "Apply"})
		case TabServices:
			hints = append(hints,
				KeyHint{"a", "Apply"},
				KeyHint{"d", "Delete"},
			)
		case TabNodes:
			hints = append(hints, KeyHint{"d", "Drain"})
		}
	case ViewDetail:
		hints = append(hints,
			KeyHint{"Esc", "Back"},
			KeyHint{"l", "Logs"},
			KeyHint{"a", "Apply"},
			KeyHint{"e", "Edit"},
		)
	case ViewLogs:
		hints = append(hints,
			KeyHint{"j/k", "Scroll"},
			KeyHint{"Enter", "Expand"},
			KeyHint{"f", "Follow"},
			KeyHint{"/", "Filter"},
			KeyHint{"Esc", "Back"},
		)
	case ViewLogDetail:
		hints = append(hints,
			KeyHint{"Esc", "Back"},
			KeyHint{"j/k", "Next/Prev"},
		)
	case ViewFilter:
		hints = append(hints,
			KeyHint{"Enter", "Apply"},
			KeyHint{"Esc", "Cancel"},
		)
	case ViewCommand:
		hints = append(hints,
			KeyHint{"Enter", "Exec"},
			KeyHint{"Esc", "Cancel"},
		)
	}

	// Globais (sempre presentes)
	hints = append(hints,
		KeyHint{"?", "Help"},
		KeyHint{"r", "Refresh"},
		KeyHint{"q", "Quit"},
	)

	return hints
}
