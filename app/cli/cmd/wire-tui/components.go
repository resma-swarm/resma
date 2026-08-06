package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ===========================================================================
// Table Component — tabela reutilizável com header, cursor e colunas flex.
// ===========================================================================

// TableColumn define uma coluna da tabela.
type TableColumn struct {
	Title string
	Width int // 0 = flex (recebe o espaço restante)
	Align lipgloss.Position
	Flex  bool // se true, recebe espaço restante proporcional
}

// TableRow representa uma linha da tabela.
type TableRow struct {
	Cells    []string // conteúdo já estilizado (com ANSI) de cada coluna
	Plain    []string // conteúdo sem ANSI (para linha selecionada)
	Selected bool
}

// TableModel é um componente de tabela reutilizável.
type TableModel struct {
	cols   []TableColumn
	rows   []TableRow
	cursor int
	width  int
	height int
	header string // título opcional acima das colunas
}

// NewTable cria uma nova tabela com as colunas especificadas.
func NewTable(cols []TableColumn) TableModel {
	return TableModel{cols: cols, cursor: 0}
}

// SetWidth define a largura total e recalcula colunas flex.
func (t *TableModel) SetWidth(w int) {
	t.width = w
	t.recalcCols()
}

// SetHeight define a altura total (número visível de linhas).
func (t *TableModel) SetHeight(h int) {
	t.height = h
}

// SetRows substitui todas as linhas.
func (t *TableModel) SetRows(rows []TableRow) {
	t.rows = rows
	if t.cursor >= len(rows) {
		t.cursor = len(rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// SetHeader define um título opcional acima das colunas.
func (t *TableModel) SetHeader(title string) {
	t.header = title
}

// Cursor retorna a posição atual do cursor.
func (t *TableModel) Cursor() int {
	return t.cursor
}

// MoveDown move o cursor para baixo (loop infinito).
func (t *TableModel) MoveDown() {
	n := len(t.rows)
	if n == 0 {
		return
	}
	t.cursor = (t.cursor + 1) % n
}

// MoveUp move o cursor para cima (loop infinito).
func (t *TableModel) MoveUp() {
	n := len(t.rows)
	if n == 0 {
		return
	}
	t.cursor = (t.cursor - 1 + n) % n
}

// MoveTop vai para a primeira linha.
func (t *TableModel) MoveTop() {
	t.cursor = 0
}

// MoveBottom vai para a última linha.
func (t *TableModel) MoveBottom() {
	if len(t.rows) > 0 {
		t.cursor = len(t.rows) - 1
	}
}

// recalcCols recalcula larguras de colunas flex para preencher o width total.
func (t *TableModel) recalcCols() {
	if t.width <= 0 {
		return
	}
	// Calcular largura fixa usada + espaços entre colunas
	fixedUsed := 0
	flexCount := 0
	for _, c := range t.cols {
		if c.Flex || c.Width == 0 {
			flexCount++
		} else {
			fixedUsed += c.Width
		}
	}
	spaces := len(t.cols) - 1 // espaços entre colunas
	avail := t.width - fixedUsed - spaces
	if avail < 0 {
		avail = 0
	}
	// Distribuir espaço disponível entre colunas flex
	if flexCount > 0 {
		flexW := avail / flexCount
		remaining := avail % flexCount
		for i := range t.cols {
			if t.cols[i].Flex || t.cols[i].Width == 0 {
				t.cols[i].Width = flexW
				if remaining > 0 {
					t.cols[i].Width++
					remaining--
				}
			}
		}
	}
}

// View renderiza a tabela completa.
func (t TableModel) View() string {
	if t.width <= 0 {
		return ""
	}

	var sb strings.Builder

	// Header opcional
	if t.header != "" {
		sb.WriteString(lipgloss.NewStyle().Width(t.width).Render(t.header))
		sb.WriteString("\n")
	}

	// Linha de cabeçalho de colunas
	var headerCells []string
	for _, c := range t.cols {
		headerCells = append(headerCells, lipgloss.NewStyle().
			Width(c.Width).
			AlignHorizontal(c.Align).
			Render(sTableHeader.Render(c.Title)))
	}
	headerLine := strings.Join(headerCells, " ")
	sb.WriteString(lipgloss.NewStyle().Width(t.width).Render(headerLine))
	sb.WriteString("\n")

	// Linhas de dados
	n := len(t.rows)
	if n == 0 {
		sb.WriteString(sMuted.Render(" No data"))
		return sb.String()
	}

	// Calcular viewport se height estiver definido
	startIdx := 0
	endIdx := n
	maxRows := n
	if t.height > 0 {
		// -2 para header de colunas + título
		maxRows = t.height - 2
		if t.header != "" {
			maxRows--
		}
		if maxRows < 3 {
			maxRows = 3
		}
		if n > maxRows {
			// Centralizar cursor no viewport
			startIdx = t.cursor - maxRows/2
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx = startIdx + maxRows
			if endIdx > n {
				endIdx = n
				startIdx = endIdx - maxRows
				if startIdx < 0 {
					startIdx = 0
				}
			}
		}
	}

	for i := startIdx; i < endIdx; i++ {
		row := t.rows[i]
		isSelected := i == t.cursor

		// Para linha selecionada, usar plain text (sem ANSI de cor)
		// e aplicar cursor background
		var cells []string
		for ci, c := range t.cols {
			var cellContent string
			if isSelected && ci < len(row.Plain) {
				cellContent = row.Plain[ci]
			} else if ci < len(row.Cells) {
				cellContent = row.Cells[ci]
			}
			// Truncar com "…" se exceder a largura da coluna (ANSI-aware)
			cellContent = truncateAnsi(cellContent, c.Width)
			cells = append(cells, lipgloss.NewStyle().
				Width(c.Width).
				AlignHorizontal(c.Align).
				Render(cellContent))
		}
		line := strings.Join(cells, " ")

		if isSelected {
			sb.WriteString(sTableCursor.Width(t.width).Render(line))
		} else {
			sb.WriteString(lipgloss.NewStyle().Width(t.width).Render(line))
		}
		sb.WriteString("\n")
	}

	// Preencher linhas vazias
	rendered := endIdx - startIdx
	for i := rendered; i < maxRows; i++ {
		sb.WriteString("\n")
	}

	return sb.String()
}

// ===========================================================================
// Popup Component — dialog overlay centralizado, sem background opaque.
// ===========================================================================

// PopupModel é um componente de popup/dialog reutilizável.
type PopupModel struct {
	title   string
	content string
	width   int
	height  int
	visible bool
}

// NewPopup cria um novo popup.
func NewPopup(title string) PopupModel {
	return PopupModel{title: title}
}

// SetContent define o conteúdo do popup.
func (p *PopupModel) SetContent(content string) {
	p.content = content
}

// SetSize define largura e altura do popup.
func (p *PopupModel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// Show exibe o popup.
func (p *PopupModel) Show() {
	p.visible = true
}

// Hide esconde o popup.
func (p *PopupModel) Hide() {
	p.visible = false
}

// IsVisible retorna se o popup está visível.
func (p *PopupModel) IsVisible() bool {
	return p.visible
}

// Render renderiza o popup sobre um background.
// O popup NÃO define background color — o terminal default mostra através,
// dando efeito de overlay sem esconder completamente o fundo.
func (p PopupModel) Render(bgWidth, bgHeight int) string {
	if !p.visible || p.content == "" {
		return ""
	}

	w := p.width
	h := p.height
	if w <= 0 {
		w = bgWidth * 80 / 100
	}
	if h <= 0 {
		h = bgHeight * 60 / 100
	}
	if w > bgWidth-2 {
		w = bgWidth - 2
	}
	if h > bgHeight-2 {
		h = bgHeight - 2
	}

	// Construir conteúdo: título + separador + body
	var sb strings.Builder
	sb.WriteString(sClusterTitle.Render(" " + p.title + " "))
	sb.WriteString("\n")
	sb.WriteString(sMuted.Render(strings.Repeat("─", w-4)))
	sb.WriteString("\n")
	sb.WriteString(p.content)

	// Popup com borda arredondada, SEM background color (overlay transparente)
	popup := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cResmaPrimary).
		Padding(0, 1).
		Render(sb.String())

	// Centralizar sobre a área de background
	return lipgloss.Place(bgWidth, bgHeight, lipgloss.Center, lipgloss.Center, popup)
}

// wrapText quebra texto em linhas de no máximo width chars (por palavra).
func wrapText(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) <= width {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	lines = append(lines, current)
	return lines
}

// truncateAnsi trunca uma string (que pode conter ANSI escape codes) para
// no máximo width chars visíveis, adicionando "…" se truncada.
// Usa lipgloss.Width para medir largura visual (ignora ANSI).
func truncateAnsi(s string, width int) string {
	if width <= 0 {
		return s
	}
	visW := lipgloss.Width(s)
	if visW <= width {
		return s
	}
	// Truncar: precisamos remover chars do final até caber + "…"
	// Como a string pode ter ANSI, precisamos iterar removendo runes do final
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if lipgloss.Width(string(runes))+1 <= width { // +1 for "…"
			return string(runes) + "…"
		}
	}
	return "…"
}
