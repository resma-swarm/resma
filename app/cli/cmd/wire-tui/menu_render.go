package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderMenu renderiza o grid de keyhints estilo k9s.
// Grid de até 6 linhas, colunas organizadas automaticamente.
func renderMenu(m model) string {
	hints := menuHints(m)

	const maxRows = 6
	n := len(hints)
	colCount := (n / maxRows)
	if n%maxRows != 0 {
		colCount++
	}

	// Distribuir hints em colunas
	cols := make([][]KeyHint, colCount)
	for i, h := range hints {
		col := i / maxRows
		cols[col] = append(cols[col], h)
	}

	// Calcular largura máxima de key por coluna
	maxKeyW := make([]int, colCount)
	for c, col := range cols {
		for _, h := range col {
			if len(h.Key) > maxKeyW[c] {
				maxKeyW[c] = len(h.Key)
			}
		}
	}

	// Renderizar grid linha por linha
	var rows []string
	for row := 0; row < maxRows; row++ {
		var parts []string
		for c := 0; c < colCount; c++ {
			if row >= len(cols[c]) {
				// padding vazio
				parts = append(parts, strings.Repeat(" ", maxKeyW[c]+12))
				continue
			}
			h := cols[c][row]
			keyW := maxKeyW[c]
			keyStyled := sMenuKey.Render(fmt.Sprintf("%-*s", keyW, h.Key))
			desc := sMenuDesc.Render(h.Desc)
			parts = append(parts, fmt.Sprintf(" %s %s", keyStyled, desc))
		}
		rows = append(rows, strings.Join(parts, "  "))
	}

	return strings.Join(rows, "\n")
}

// Estilos do menu
var (
	sMenuKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(cAccent)

	sMenuDesc = lipgloss.NewStyle().
			Foreground(cMuted)
)
