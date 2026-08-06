# RESMA CLI — Estudo: Tabelas Responsivas no TUI

> **Status:** Pesquisa técnica concluída
> **Data:** 2026-08-06
> **Questão:** Como fazer tabelas que ocupam todo o espaço horizontal E adaptam o tamanho das colunas aos dados?

---

## 1. O Problema

O `bubbles/table` (componente padrão do Bubble Tea para tabelas) tem **widths de coluna fixos** definidos na construção:

```go
columns := []table.Column{
    {Title: "NAME", Width: 20},     // fixo em 20 chars
    {Title: "CPU%",  Width: 8},      // fixo em 8 chars
    {Title: "MEM%",  Width: 8},      // fixo em 8 chars
}
```

### 1.1 Limitações do bubbles/table

| Limitação | Impacto |
|-----------|---------|
| **Widths fixos** | Colunas não se adaptam ao conteúdo (nome longo é truncado, nome curto desperdiça espaço) |
| `SetWidth()` só muda o viewport | A largura externa muda, mas as colunas internas não se redimensionam — aparece espaço vazio à direita ou borda direita ausente |
| **Sem auto-fit ao terminal** | Redimensionar o terminal não recalcula as colunas — a tabela fica com tamanho errado |
| **Sem auto-size por conteúdo** | Não há algoritmo para calcular width ideal baseado nos dados |
| Issue [#894](https://github.com/charmbracelet/bubbles/issues/894) aberto desde 2023 | Maintainers disseram que planejam migrar para lipgloss/table, mas ainda não fizeram |

### 1.2 O que precisamos

1. **Ocupar 100% da largura horizontal** disponível (main panel width)
2. **Auto-size das colunas** baseado no conteúdo (mediana/percentil dos valores)
3. **Redistribuir espaço extra** para colunas flexíveis (ex: NAME, STATUS)
4. **Respeitar widths mínimos** para colunas numéricas (ex: CPU%, MEM%)
5. **Truncar com ellipsis** quando conteúdo excede o width disponível
6. **Recalcular on resize** (tea.WindowSizeMsg)

---

## 2. Análise das Alternativas

### 2.1 lipgloss/table (v1.1.0+)

O lipgloss/table tem um algoritmo sofisticado de resize (`table/resizing.go`):

**Algoritmo:**
1. Calcula `min`, `max`, `median` do conteúdo de cada coluna
2. Se a tabela é **mais estreita** que o width especificado → **expande** colunas uniformemente
3. Se a tabela é **mais larga** que o width especificado → **encolhe** colunas baseado na maior diferença entre median e width (preserva dados importantes)
4. **Wrap de conteúdo** por default (v1.1.0+)

**Exemplo do algoritmo de shrink:**
```
┌──────┬───────────────┬──────────┐
│ Name │ Age of Person │ Location │
├──────┼───────────────┼──────────┤
│ Kini │ 40            │ New York │
│ Eli  │ 30            │ London   │
│ Iris │ 20            │ Paris    │
└──────┴───────────────┴──────────┘

Median non-whitespace length vs column width:
  Name:           4 / 5    → diff = 1
  Age of Person:  2 / 15   → diff = 13  ← maior diferença, encolhe 13
  Location:       6 / 10   → diff = 4
```

**Prós:**
- Auto-sizing sofisticado baseado em mediana
- Wrap de conteúdo
- Ocupa exatamente o width especificado

**Contras:**
- **Não é um componente Bubble Tea** — é apenas um renderer (sem cursor, sem KeyMap, sem scroll, sem selection)
- Não serve para tabelas interativas com navegação j/k
- Ideal para output estático (ex: `resma services list` em modo não-TUI)

### 2.2 bubbles/table com auto-sizing manual

**Padrão:** usar bubbles/table (para navegação/cursor/KeyMap) mas recalcular os widths das colunas manualmente a cada `tea.WindowSizeMsg`.

**Prós:**
- Mantém cursor, KeyMap, scroll, selection, Focus/Blur
- Controle total sobre o algoritmo de sizing
- Funciona com o componente já escolhido

**Contras:**
- Requer implementar o algoritmo de auto-sizing nós mesmos
- Precisa recalcular a cada resize e a cada mudança de dados

### 2.3 tview.Table (k9s)

O k9s usa `tview.Table` (da biblioteca tview, não Bubble Tea) que tem:
- Column management com hide/show
- Wide mode (toggle com tecla)
- Custom columns via `views.yaml`
- Sortable columns

**Contras:**
- tview é um framework diferente (rivo/tview) — não integra com Bubble Tea
- Adotar tview significaria abandonar toda a stack Charmbracelet

### 2.4 Decisão

**Usar bubbles/table com auto-sizing manual** (opção 2.2).

Motivos:
1. Mantemos toda a stack Charmbracelet (Bubble Tea + Lipgloss + Bubbles)
2. Preservamos cursor, KeyMap, scroll, selection, Focus/Blur
3. Controle total sobre o algoritmo — podemos adaptar para as necessidades do RESMA
4. O algoritmo de auto-sizing não é complexo — ~100 linhas de Go

---

## 3. Algoritmo de Auto-Sizing Proposto

### 3.1 Conceito: ColumnSpec

Cada coluna tem uma spec que define seu comportamento de sizing:

```go
type ColumnSpec struct {
    Title    string
    MinWidth int       // width mínimo (para colunas numéricas: CPU%, MEM%)
    MaxWidth int       // width máximo (para colunas de nome: 40 chars)
    Flex     float64   // peso para redistribuição de espaço extra (0 = fixa, 1 = flexível)
    Align    Alignment // left (default) | right | center
}

type Alignment int

const (
    AlignLeft Alignment = iota
    AlignRight
    AlignCenter
)
```

### 3.2 Algoritmo — 3 fases

```
FASE 1: Calcular content width de cada coluna
  para cada coluna:
    content_width = max(len(header), percentile90(valores))
    se content_width < min_width: content_width = min_width
    se content_width > max_width: content_width = max_width

FASE 2: Verificar se cabe no available_width
  total_content = soma(content_widths) + (num_cols - 1) * separator_width
  se total_content <= available_width:
    // EXPANDIR — redistribuir espaço extra
    extra = available_width - total_content
    para cada coluna com flex > 0:
      coluna.width += extra * (coluna.flex / soma_flex)
  senão:
    // ENCOLHER — reduzir colunas flexíveis proporcionalmente
    deficit = total_content - available_width
    para cada coluna com flex > 0 (em ordem decrescente de flex):
      redução = min(deficit, coluna.width - coluna.min_width)
      coluna.width -= redução
      deficit -= redução
      se deficit == 0: break

FASE 3: Aplicar aos componentes
  para cada coluna:
    table.Column{Title: spec.Title, Width: coluna.width}
  chamar table.SetColumns() com as novas colunas
```

### 3.3 Por que percentil 90 (não mediana)?

- **Mediana** (lipgloss): bom para shrink, mas ignora outliers
- **Percentil 90**: a maioria dos valores cabe sem truncamento, apenas 10% dos outliers são truncados com `…`
- Para RESMA, nomes de serviços são geralmente curtos (`api`, `worker-3`), mas occasionalmente longos (`ml-inference-gpu-batch`)
- Percentil 90 garante que 90% dos nomes apareçam inteiros

### 3.4 Exemplo prático

**Cenário:** Tab Services, terminal 120 chars, main panel 67% = ~76 chars

```
Available width: 76 chars
Separator: 1 char entre colunas (3 separadores = 3 chars)
Columns:
  NAME     — min 10, max 40, flex 1.0  → content = p90(services) = 18
  REPLICAS — min 8, max 8,  flex 0.0  → content = 8 (fixo)
  CPU%     — min 6, max 6,  flex 0.0  → content = 6 (fixo)
  MEM%     — min 6, max 6,  flex 0.0  → content = 6 (fixo)
  STATUS   — min 8, max 12, flex 0.5  → content = 8
  TREND    — min 15, max 25, flex 0.5 → content = 20

Fase 1: content widths = [18, 8, 6, 6, 8, 20] = 66
Fase 2: total = 66 + 5 (separadores) = 71 <= 76 → EXPANDIR
  extra = 76 - 71 = 5
  soma_flex = 1.0 + 0.5 + 0.5 = 2.0
  NAME:   18 + 5 * (1.0/2.0) = 18 + 2 = 20
  STATUS: 8  + 5 * (0.5/2.0) = 8  + 1 = 9
  TREND:  20 + 5 * (0.5/2.0) = 20 + 1 = 21
  (arredondamento: 2+1+1 = 4, sobra 1 → vai para NAME por ter maior flex)

Resultado: [21, 8, 6, 6, 9, 21] = 71 chars + 5 separadores = 76 ✓
```

---

## 4. Implementação de Referência

### 4.1 AutoTable — wrapper sobre bubbles/table

```go
package tui

import (
	"sort"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AutoTable é um wrapper sobre bubbles/table que recalcula
// os widths das colunas automaticamente baseado em:
//   1. Largura disponível do container
//   2. Conteúdo dos dados (percentil 90)
//   3. Spec de cada coluna (min/max/flex)
type AutoTable struct {
	table.Model
	specs       []ColumnSpec
	rows        []table.Row
	availableW  int
	borderWidth int // largura das bordas verticais (geralmente 2)
}

// NewAutoTable cria uma nova AutoTable com as specs de coluna.
func NewAutoTable(specs []ColumnSpec) *AutoTable {
	t := &AutoTable{
		specs:       specs,
		borderWidth: 2, // borda esquerda + direita
	}
	// Inicializa com widths mínimos
	cols := make([]table.Column, len(specs))
	for i, s := range specs {
		cols[i] = table.Column{Title: s.Title, Width: s.MinWidth}
	}
	t.Model = table.New(table.WithColumns(cols), table.WithFocused(true))
	return t
}

// SetRows define as linhas e recalcula os widths das colunas.
func (t *AutoTable) SetRows(rows []table.Row) {
	t.rows = rows
	t.recalculateColumns()
	t.Model.SetRows(rows)
}

// SetWidth sobrescreve o método para recalcular colunas.
func (t *AutoTable) SetWidth(w int) {
	t.availableW = w
	t.recalculateColumns()
	t.Model.SetWidth(w)
}

// recalculateColumns recalcula os widths das colunas baseado em:
//   - conteúdo dos dados (percentil 90)
//   - largura disponível
//   - specs (min/max/flex)
func (t *AutoTable) recalculateColumns() {
	if len(t.specs) == 0 {
		return
	}

	numCols := len(t.specs)
	separatorWidth := numCols - 1 // 1 char entre cada coluna
	availableForCols := t.availableW - t.borderWidth - separatorWidth
	if availableForCols <= 0 {
		// Terminal muito pequeno — usar mínimos
		cols := make([]table.Column, numCols)
		for i, s := range t.specs {
			cols[i] = table.Column{Title: s.Title, Width: s.MinWidth}
		}
		t.SetColumns(cols)
		return
	}

	// FASE 1: Calcular content width de cada coluna
	contentWidths := make([]int, numCols)
	for i, spec := range t.specs {
		cw := t.calculateContentWidth(i, spec)
		contentWidths[i] = clamp(cw, spec.MinWidth, spec.MaxWidth)
	}

	// FASE 2: Verificar se cabe
	totalContent := sum(contentWidths)
	if totalContent <= availableForCols {
		// EXPANDIR — redistribuir espaço extra
		t.expandColumns(contentWidths, availableForCols-totalContent)
	} else {
		// ENCOLHER — reduzir colunas flexíveis
		t.shrinkColumns(contentWidths, totalContent-availableForCols)
	}

	// FASE 3: Aplicar
	cols := make([]table.Column, numCols)
	for i, spec := range t.specs {
		cols[i] = table.Column{Title: spec.Title, Width: contentWidths[i]}
	}
	t.SetColumns(cols)
}

// calculateContentWidth calcula o width ideal baseado no percentil 90 dos valores.
func (t *AutoTable) calculateContentWidth(colIdx int, spec ColumnSpec) int {
	// Header width
	headerWidth := lipgloss.Width(spec.Title)

	if len(t.rows) == 0 {
		return headerWidth
	}

	// Coletar widths de todas as células da coluna
	widths := make([]int, 0, len(t.rows))
	for _, row := range t.rows {
		if colIdx < len(row) {
			widths = append(widths, lipgloss.Width(row[colIdx]))
		}
	}

	if len(widths) == 0 {
		return headerWidth
	}

	// Percentil 90
	p90 := percentile(widths, 90)

	return max(headerWidth, p90)
}

// expandColumns redistribui espaço extra para colunas flexíveis.
func (t *AutoTable) expandColumns(widths []int, extra int) {
	if extra <= 0 {
		return
	}

	totalFlex := 0.0
	for _, s := range t.specs {
		totalFlex += s.Flex
	}

	if totalFlex == 0 {
		// Sem colunas flexíveis — distribuir igualmente
		perCol := extra / len(widths)
		for i := range widths {
			widths[i] += perCol
		}
		return
	}

	// Distribuir proporcionalmente ao flex
	remaining := extra
	for i, s := range t.specs {
		if s.Flex <= 0 {
			continue
		}
		share := int(float64(extra) * (s.Flex / totalFlex))
		// Não exceder MaxWidth
		share = min(share, s.MaxWidth-widths[i])
		widths[i] += share
		remaining -= share
	}

	// Distribuir remainder (arredondamento) para a coluna com maior flex
	if remaining > 0 {
		maxFlexIdx := 0
		maxFlex := 0.0
		for i, s := range t.specs {
			if s.Flex > maxFlex && widths[i] < s.MaxWidth {
				maxFlex = s.Flex
				maxFlexIdx = i
			}
		}
		widths[maxFlexIdx] += remaining
	}
}

// shrinkColumns reduz colunas flexíveis proporcionalmente ao deficit.
func (t *AutoTable) shrinkColumns(widths []int, deficit int) {
	if deficit <= 0 {
		return
	}

	// Ordenar colunas por flex decrescente (mais flexíveis encolhem primeiro)
	indices := make([]int, len(t.specs))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(a, b int) bool {
		return t.specs[indices[a]].Flex > t.specs[indices[b]].Flex
	})

	for _, idx := range indices {
		if deficit <= 0 {
			break
		}
		spec := t.specs[idx]
		if spec.Flex <= 0 {
			continue
		}
		// O quanto podemos reduzir sem passar do MinWidth
		reducible := widths[idx] - spec.MinWidth
		reduction := min(deficit, reducible)
		widths[idx] -= reduction
		deficit -= reduction
	}
}

// ─── Helpers ─────────────────────────────────────────────

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sum(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

// percentile calcula o percentil p de uma lista de inteiros.
func percentile(sorted []int, p int) int {
	if len(sorted) == 0 {
		return 0
	}
	// Cópia para não mutar o original
	xs := make([]int, len(sorted))
	copy(xs, sorted)
	sort.Ints(xs)

	// Índice do percentil (nearest-rank method)
	idx := (p * len(xs)) / 100
	if idx >= len(xs) {
		idx = len(xs) - 1
	}
	return xs[idx]
}
```

### 4.2 Uso no DashboardModel

```go
// No calculateLayout(), quando receber WindowSizeMsg:
func (m *DashboardModel) calculateLayout() {
	// ... calcular availableWidth ...

	// Recalcular largura da tabela
	mainWidth := availableWidth - sideWidth
	m.serviceTable.SetWidth(mainWidth - 2) // -2 para bordas
}

// No Init ou quando dados chegam via SSE:
func (m *DashboardModel) updateServices(services []ServiceMetrics) {
	rows := make([]table.Row, len(services))
	for i, s := range services {
		rows[i] = table.Row{
			s.Name,
			fmt.Sprintf("%d/%d", s.ReplicasReady, s.ReplicasTotal),
			fmt.Sprintf("%.1f", s.CPU),
			fmt.Sprintf("%.1f", s.Mem),
			string(s.Status),
			renderSparkline(s.Sparkline, 20),
		}
	}
	m.serviceTable.SetRows(rows) // AutoTable recalcula colunas automaticamente
}
```

### 4.3 Specs por tab

```go
// Services Tab
var serviceColumnSpecs = []ColumnSpec{
	{Title: "NAME",     MinWidth: 10, MaxWidth: 40, Flex: 1.0, Align: AlignLeft},
	{Title: "REPLICAS", MinWidth: 8,  MaxWidth: 8,  Flex: 0.0, Align: AlignRight},
	{Title: "CPU%",     MinWidth: 6,  MaxWidth: 6,  Flex: 0.0, Align: AlignRight},
	{Title: "MEM%",     MinWidth: 6,  MaxWidth: 6,  Flex: 0.0, Align: AlignRight},
	{Title: "STATUS",   MinWidth: 8,  MaxWidth: 12, Flex: 0.5, Align: AlignLeft},
	{Title: "TREND",    MinWidth: 15, MaxWidth: 25, Flex: 0.5, Align: AlignLeft},
}

// Nodes Tab
var nodeColumnSpecs = []ColumnSpec{
	{Title: "NAME",     MinWidth: 10, MaxWidth: 30, Flex: 1.0, Align: AlignLeft},
	{Title: "ROLE",     MinWidth: 6,  MaxWidth: 10, Flex: 0.3, Align: AlignLeft},
	{Title: "CPU%",     MinWidth: 6,  MaxWidth: 6,  Flex: 0.0, Align: AlignRight},
	{Title: "MEM%",     MinWidth: 6,  MaxWidth: 6,  Flex: 0.0, Align: AlignRight},
	{Title: "DISK%",    MinWidth: 7,  MaxWidth: 7,  Flex: 0.0, Align: AlignRight},
	{Title: "SERVICES", MinWidth: 8,  MaxWidth: 8,  Flex: 0.0, Align: AlignRight},
	{Title: "STATUS",   MinWidth: 8,  MaxWidth: 10, Flex: 0.3, Align: AlignLeft},
}

// Tasks Tab
var taskColumnSpecs = []ColumnSpec{
	{Title: "ID",       MinWidth: 8,  MaxWidth: 12, Flex: 0.5, Align: AlignLeft},
	{Title: "SERVICE",  MinWidth: 10, MaxWidth: 30, Flex: 1.0, Align: AlignLeft},
	{Title: "NODE",     MinWidth: 8,  MaxWidth: 20, Flex: 0.5, Align: AlignLeft},
	{Title: "STATUS",   MinWidth: 8,  MaxWidth: 12, Flex: 0.3, Align: AlignLeft},
	{Title: "UPTIME",   MinWidth: 8,  MaxWidth: 10, Flex: 0.0, Align: AlignRight},
	{Title: "RESTARTS", MinWidth: 8,  MaxWidth: 8,  Flex: 0.0, Align: AlignRight},
}

// Alerts Tab
var alertColumnSpecs = []ColumnSpec{
	{Title: "TIME",     MinWidth: 8,  MaxWidth: 10, Flex: 0.0, Align: AlignLeft},
	{Title: "LEVEL",    MinWidth: 8,  MaxWidth: 10, Flex: 0.0, Align: AlignLeft},
	{Title: "SERVICE",  MinWidth: 10, MaxWidth: 30, Flex: 1.0, Align: AlignLeft},
	{Title: "MESSAGE",  MinWidth: 20, MaxWidth: 60, Flex: 1.0, Align: AlignLeft},
}

// Recommendations Tab
var recColumnSpecs = []ColumnSpec{
	{Title: "SERVICE",  MinWidth: 10, MaxWidth: 30, Flex: 1.0, Align: AlignLeft},
	{Title: "RISK",     MinWidth: 8,  MaxWidth: 10, Flex: 0.0, Align: AlignLeft},
	{Title: "TIER",     MinWidth: 10, MaxWidth: 15, Flex: 0.3, Align: AlignLeft},
	{Title: "CPU",      MinWidth: 10, MaxWidth: 10, Flex: 0.0, Align: AlignLeft},
	{Title: "MEM",      MinWidth: 10, MaxWidth: 10, Flex: 0.0, Align: AlignLeft},
	{Title: "REASON",   MinWidth: 20, MaxWidth: 50, Flex: 1.0, Align: AlignLeft},
}
```

### 4.4 Alinhamento de colunas

O `bubbles/table` não suporta alinhamento por coluna nativamente. Para alinhar à direita colunas numéricas, pré-formatar o valor com `fmt.Sprintf`:

```go
func formatRight(s string, width int) string {
	if lipgloss.Width(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-lipgloss.Width(s)) + s
}

// Na construção de rows:
rows[i] = table.Row{
	s.Name,                                           // left
	formatRight(fmt.Sprintf("%d/%d", ready, total), 8), // right
	formatRight(fmt.Sprintf("%.1f", cpu), 6),           // right
	formatRight(fmt.Sprintf("%.1f", mem), 6),           // right
	string(s.Status),                                  // left
	renderSparkline(s.Sparkline, 20),                  // left
}
```

Alternativa mais elegante: usar `lipgloss.Style` com `Align`:

```go
func alignCell(s string, width int, align Alignment) string {
	style := lipgloss.NewStyle().Width(width).MaxWidth(width)
	switch align {
	case AlignRight:
		style = style.AlignHorizontal(lipgloss.Right)
	case AlignCenter:
		style = style.AlignHorizontal(lipgloss.Center)
	}
	return style.Render(s)
}
```

> **Cuidado:** `lipgloss.Style.Render()` adiciona ANSI codes que afetam `lipgloss.Width()`.
> Ao calcular content width, usar `lipgloss.Width()` (que desconta ANSI) e não `len()`.

---

## 5. Edge Cases

### 5.1 Terminal muito pequeno

Quando `availableForCols <= 0`, usar `MinWidth` de todas as colunas. Se mesmo com MinWidth não couber, a tabela vai transbordar (o viewport do bubbles/table faz scroll horizontal).

### 5.2 Poucas linhas (lista vazia ou 1 item)

Quando `len(rows) == 0`, usar `headerWidth` como content width. Quando `len(rows) == 1`, o percentil 90 é o próprio valor — funciona corretamente.

### 5.3 Conteúdo com ANSI codes (sparklines coloridas)

`lipgloss.Width()` desconta ANSI codes corretamente (usa `ansi.StringWidth`). Sempre usar `lipgloss.Width()` em vez de `len()` ou `utf8.RuneCountInString()`.

### 5.4 Conteúdo muito largo (URLs, mensagens de erro)

O `MaxWidth` da spec limita o tamanho máximo. O `bubbles/table` trunca com `…` automaticamente:

```go
// Interno do bubbles/table:
style := lipgloss.NewStyle().Width(col.Width).MaxWidth(col.Width).Inline(true)
renderedCell := style.Render(ansi.Truncate(value, col.Width, "…"))
```

### 5.5 Resize frequente

O recálculo é O(n * m) onde n = linhas e m = colunas. Para 100 linhas e 6 colunas = 600 operações — trivial. Não há necessidade de debounce.

### 5.6 Dados atualizando via SSE

A cada evento SSE que atualiza os dados, `SetRows()` é chamado, que dispara `recalculateColumns()`. Se os dados mudaram (ex: novo serviço com nome mais longo), as colunas se adaptam automaticamente.

---

## 6. Comparação: lipgloss/table vs bubbles/table + AutoTable

| Critério | lipgloss/table | bubbles/table + AutoTable |
|----------|----------------|--------------------------|
| Auto-sizing | ✅ Algoritmo sofisticado (mediana) | ✅ Algoritmo custom (p90 + flex) |
| Ocupa 100% width | ✅ | ✅ |
| Wrap de conteúdo | ✅ (v1.1.0+) | ❌ (trunca com `…`) |
| Cursor/navegação | ❌ | ✅ |
| KeyMap (j/k, PageUp/Down) | ❌ | ✅ |
| Selection highlight | ❌ | ✅ |
| Focus/Blur | ❌ | ✅ |
| Scroll vertical | ❌ | ✅ (via viewport interno) |
| Integração Bubble Tea | ❌ (é renderer) | ✅ (é componente) |
| Ideal para | Output estático (`resma services list`) | TUI interativo (`resma monitor`) |

**Decisão:** Usar `bubbles/table + AutoTable` para o TUI interativo (`resma monitor`).
Reservar `lipgloss/table` para output estático de comandos CLI não-TUI (ex: `resma services list --format table`).

---

## 7. Atualização na Spec

Este estudo deve ser integrado ao `tui-design.md` como uma nova seção "Tabelas Responsivas" substituindo a seção atual de ServiceTable, e a struct `AutoTable` deve fazer parte da estrutura de arquivos TUI.

### 7.1 Novo arquivo na estrutura TUI

```
app/cli/internal/tui/
├── components/
│   ├── autotable.go          # AutoTable wrapper + ColumnSpec + algoritmo de auto-sizing
│   ├── servicetable.go       # Usa AutoTable com serviceColumnSpecs
│   ├── nodestable.go         # Usa AutoTable com nodeColumnSpecs
│   ├── taskstable.go         # Usa AutoTable com taskColumnSpecs
│   ├── alertstable.go        # Usa AutoTable com alertColumnSpecs
│   └── recommendationstable.go # Usa AutoTable com recColumnSpecs
```

### 7.2 lipgloss/table para output CLI estático

Para comandos não-TUI (ex: `resma services list`), usar `lipgloss/table` que tem auto-sizing nativo:

```go
import "github.com/charmbracelet/lipgloss/table"

func renderServicesTable(services []ServiceMetrics, width int) string {
	t := table.New().
		Headers("NAME", "REPLICAS", "CPU%", "MEM%", "STATUS").
		Border(lipgloss.NormalBorder()).
		Width(width)

	for _, s := range services {
		t.Row(s.Name, fmt.Sprintf("%d/%d", s.ReplicasReady, s.ReplicasTotal),
			fmt.Sprintf("%.1f", s.CPU), fmt.Sprintf("%.1f", s.Mem), string(s.Status))
	}

	return t.String()
}
```

O `lipgloss/table` com `.Width(width)` faz o auto-sizing automaticamente usando o algoritmo de mediana descrito na seção 2.1.
