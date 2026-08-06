package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// rowCell renderiza uma célula com largura fixa (ANSI-aware via lipgloss.Width).
func rowCell(text string, width int, align lipgloss.Position) string {
	return lipgloss.NewStyle().Width(width).AlignHorizontal(align).Render(text)
}

func cellLeft(text string, width int) string {
	return rowCell(text, width, lipgloss.Left)
}

func cellRight(text string, width int) string {
	return rowCell(text, width, lipgloss.Right)
}

// joinRow junta células horizontalmente sem padding extra.
func joinRow(cells ...string) string {
	return strings.Join(cells, " ")
}

// sparklinePlain retorna sparkline sem cores (para linha selecionada).
func sparklinePlain(data []float64) string {
	if len(data) == 0 {
		return ""
	}
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	result := make([]rune, 0, len(data))
	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return ""
	}
	for _, v := range data {
		idx := int((v / maxVal) * float64(len(chars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(chars) {
			idx = len(chars) - 1
		}
		result = append(result, chars[idx])
	}
	return string(result)
}

// renderTableRow renderiza uma linha de tabela ocupando exatamente totalWidth.
// Usa lipgloss.Width (ANSI-aware) para garantir que a linha não wrap.
func renderTableRow(cells []string, totalWidth int, selected bool) string {
	row := strings.Join(cells, " ")
	style := lipgloss.NewStyle().Width(totalWidth)
	if selected {
		style = sTableCursor.Width(totalWidth)
	}
	return style.Render(row)
}

// colWidths calcula larguras de colunas dado o total e uma coluna flex.
// fixedCols = larguras fixas, flexIdx = índice da coluna que recebe o resto.
func colWidths(total int, fixedCols []int, spacing int) int {
	used := 0
	for _, w := range fixedCols {
		used += w
	}
	used += spacing * (len(fixedCols)) // espaços entre colunas
	flex := total - used
	if flex < 5 {
		flex = 5
	}
	return flex
}

// pctStr formata um float como "XX.X%".
func pctStr(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}
