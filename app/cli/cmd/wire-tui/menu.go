package main

// KeyHint representa uma keybinding exibida no menu.
type KeyHint struct {
	Key  string // ex: "0", "a", "Enter"
	Desc string // ex: "Services", "Apply"
}

// menuHints retorna as keyhints ativas para o contexto atual.
func menuHints(m model) []KeyHint {
	var hints []KeyHint

	// Tab navigation (sempre presente)
	hints = append(hints,
		KeyHint{"0", "Services"},
		KeyHint{"1", "Nodes"},
		KeyHint{"2", "Agents"},
		KeyHint{"3", "Tasks"},
		KeyHint{"4", "Alerts"},
		KeyHint{"5", "Recs"},
	)

	// Ações por view mode
	switch m.viewMode {
	case ViewList:
		hints = append(hints,
			KeyHint{"Enter", "Detail"},
			KeyHint{"l", "Logs"},
			KeyHint{"/", "Filter"},
			KeyHint{":", "Command"},
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
			KeyHint{"a", "Apply"},
			KeyHint{"e", "Edit"},
			KeyHint{"l", "Logs"},
		)
	case ViewFilter:
		hints = append(hints,
			KeyHint{"Enter", "Apply"},
			KeyHint{"Esc", "Cancel"},
		)
	case ViewCommand:
		hints = append(hints,
			KeyHint{"Enter", "Execute"},
			KeyHint{"Esc", "Cancel"},
			KeyHint{"Tab", "Complete"},
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
