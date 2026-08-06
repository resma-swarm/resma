# RESMA CLI — Design do Dashboard TUI

> **Status:** Spec completa — CLI completo (não MVP). Todas as features do Conselho Técnico incluídas.
>
> **Referências estudadas:** k9s (derailed/k9s, 25K stars), lazydocker (jesseduffield, 40K stars),
> lazygit (50K stars), Bubble Tea patterns (Charmbracelet), 4 Golden Rules de TUI Layout.
>
> **Conselho Técnico:** 3 personas (Arquiteto TUI, UX/DX Specialist, Critic) analisaram o mock
> TUI inicial e propuseram melhorias. Esta spec incorpora todas as recomendações sem cortes.

---

## 1. Visão Geral

O comando `resma monitor` abre um dashboard TUI em modo **alt-screen** que apresenta métricas
em tempo real consumidas de streams **SSE (Server-Sent Events)** emitidos pelo backend do RESMA.
A navegação é totalmente por teclado (com suporte opcional a mouse), seguindo a arquitetura Elm
do framework **Bubble Tea**.

### 1.1 Características principais

- **Layout two-column** (side panel + main panel) inspirado em lazydocker
- **6 tabs** com navegação por teclas `1-6` ou `Tab/Shift+Tab`
- **Drill-down** com view stack (Enter desce, Esc sobe) inspirado em k9s
- **Command mode** estilo k9s (`:services`, `:apply api`, `:nodes node-1`)
- **Filter mode** com regex (`/api`, `/node-[12]`, `/.*-dev`)
- **Help overlay** contextual (`?` mostra keybindings do contexto atual)
- **Skins/themes** customizáveis via YAML (estilo k9s skins)
- **HotKeys** customizáveis via config YAML
- **Mouse support** opcional (click, scroll, drag)
- **Breadcrumbs** mostrando caminho de navegação
- **Sparklines** via asciigraph, **gráficos** inline
- **Loading states** com spinner, **toast** de success/error
- **Confirmation modals** para ações destrutivas
- **Responsivo** — portrait mode, minimum size check, weights (não pixels)
- **Tema claro/escuro** detectado via termenv + colorprofile

### 1.2 Referências e inspirações

| Referência | O que aproveitamos | O que NÃO copiamos |
|------------|-------------------|-------------------|
| **k9s** | Command mode `:`, drill-down Enter/Esc, skins YAML, hotkeys, breadcrumbs, help `?` | Single-resource-view (RESMA precisa lista+detalhe), 300+ views |
| **lazydocker** | Two-column layout (side 33% + main 67%), portrait mode, accordion, bordas configuráveis | Tabs no main panel (RESMA usa tabs no topo) |
| **lazygit** | Layout calculation on resize, scroll past bottom | gocui (usamos Bubble Tea) |
| **Bubble Tea** | Elm Architecture, bubbles/table, bubbles/viewport, bubbles/list, Focus/Blur, 4 Golden Rules | — |

---

## 2. Layout

### 2.1 Estrutura — Two-Column com Tabs

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  RESMA  Docker Swarm Resource Manager              2025-01-15 14:32:01  ●    │ ← Header (1 linha)
├──────────────────────────────────────────────────────────────────────────────┤
│  [1] Services  [2] Nodes  [3] Agents  [4] Tasks  [5] Alerts  [6] Recs       │ ← Tab bar (1 linha)
├────────────────────┬─────────────────────────────────────────────────────────┤
│                    │                                                         │
│  SIDE PANEL (33%)  │  MAIN PANEL (67%)                                      │
│                    │                                                         │
│  ┌──────────────┐  │  ┌─────────────────────────────────────────────────┐  │
│  │ Filter: api_ │  │  │ NAME            REPLICAS  CPU%  MEM%  STATUS    │  │
│  │──────────────│  │  │─────────────────────────────────────────────────│  │
│  │ ► api        │  │  │▶ api            3/3       45.2  62.1  running   │  │ ← cursor (highlight)
│  │   ml-infer.. │  │  │  ml-inference   1/1       12.0  89.0  running   │  │
│  │   frontend.. │  │  │  frontend-dev   1/1        3.1  18.5  running   │  │
│  │   worker-3   │  │  │  worker-3       2/2       62.0  40.0  running   │  │
│  │   postgres   │  │  │  postgres       1/1       22.0  71.0  running   │  │
│  │   redis-ca.. │  │  │  redis-cache    2/2        8.0  35.0  running   │  │
│  │   nginx-pr.. │  │  │  nginx-proxy    3/3        5.0  12.0  running   │  │
│  │   batch-pr.. │  │  │  batch-proc..   0/2        0.0   0.0  stopped   │  │
│  └──────────────┘  │  └─────────────────────────────────────────────────┘  │
│                    │                                                         │
│                    │  ┌─ CPU Trend — api ────────────────────────────────┐  │
│                    │  │  ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂▁                  │  │
│                    │  └──────────────────────────────────────────────────┘  │
│                    │                                                         │
├────────────────────┴─────────────────────────────────────────────────────────┤
│  Services > api                                                               │ ← Breadcrumb
├──────────────────────────────────────────────────────────────────────────────┤
│  q quit · j/k move · Enter detail · / filter · : command · ? help · a apply │ ← Footer contextual
└──────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Painéis e proporções

| Painel | Largura | Conteúdo | Componente Bubble Tea |
|--------|---------|----------|----------------------|
| **Header** | 100% (1 linha) | Título + relógio + status SSE | lipgloss style |
| **Tab bar** | 100% (1 linha) | 6 tabs com highlight na ativa | lipgloss style |
| **Side panel** | 33% | Lista filtrável de itens da tab atual | `bubbles/list` |
| **Main panel** | 67% | Tabela com cursor + cards + sparklines | `bubbles/table` + `bubbles/viewport` |
| **Breadcrumb** | 100% (1 linha) | Caminho de navegação atual | lipgloss style |
| **Footer** | 100% (1 linha) | Keybindings do contexto atual | lipgloss style |

### 2.3 Portrait mode

Quando `width <= 84 && height > 45`, o layout muda para vertical (side panel em cima, main embaixo):

```
┌──────────────────────────────────────────────────────────┐
│  RESMA  Docker Swarm Resource Manager     14:32:01  ●   │
├──────────────────────────────────────────────────────────┤
│  [1] Services [2] Nodes [3] Agents [4] Tasks [5] [6]    │
├──────────────────────────────────────────────────────────┤
│  SIDE PANEL (33% height)                                │
│  ► api  │  ml-inference  │  frontend-dev  │  worker-3   │
├──────────────────────────────────────────────────────────┤
│  MAIN PANEL (67% height)                                │
│  NAME          REPLICAS  CPU%  MEM%  STATUS             │
│  ▶ api         3/3       45.2  62.1  running            │
│  ...                                                     │
├──────────────────────────────────────────────────────────┤
│  q quit · j/k move · Enter detail · / filter · ? help   │
└──────────────────────────────────────────────────────────┘
```

### 2.4 Minimum terminal size

| Dimensão | Mínimo | Comportamento abaixo do mínimo |
|----------|--------|-------------------------------|
| Width | 80 chars | Mensagem: "Terminal too small: WxH (min: 80x24). Press q to quit." |
| Height | 24 lines | Mesma mensagem |

### 2.5 Golden Rules de layout (Bubble Tea)

1. **Subtrair 2 do height/width para borders** antes de renderizar painéis
2. **Nunca auto-wrap em painéis com borda** — truncar explicitamente com `lipgloss.Width()`
3. **Match mouse detection to layout** — X coords para horizontal, Y coords para vertical
4. **Usar weights, não pixels** — proporções (0.33/0.67) em vez de valores fixos

---

## 3. Modelo — Bubble Tea (Elm Architecture)

### 3.1 DashboardModel

```go
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/spinner"
)

// ViewMode representa o modo de visualização atual do dashboard.
type ViewMode int

const (
	ViewList        ViewMode = iota // Lista de itens (default)
	ViewDetail                      // Drill-down de um item
	ViewFilter                      // Modo filtro (/)
	ViewCommand                     // Command mode (:)
	ViewHelp                        // Overlay de help (?)
	ViewConfirm                     // Modal de confirmação
)

// PanelID identifica qual painel está focado.
type PanelID int

const (
	PanelSide PanelID = iota // Side panel (lista)
	PanelMain                // Main panel (tabela/detalhe)
)

// LayoutMode identifica orientação do layout.
type LayoutMode int

const (
	LayoutLandscape LayoutMode = iota // Side-by-side (default)
	LayoutPortrait                    // Stacked vertical
)

// ViewStackEntry registra um nível de drill-down para navegação back.
type ViewStackEntry struct {
	Tab    TabID
	Cursor int
	Filter string
	Mode   ViewMode
}

// DashboardModel é o modelo raiz do dashboard TUI do `resma monitor`.
type DashboardModel struct {
	// Estado de navegação
	activeTab    TabID
	viewMode     ViewMode
	focusedPanel PanelID
	layoutMode   LayoutMode
	viewStack    []ViewStackEntry // histórico de drill-down

	// Estado de lista
	cursor   int    // índice do item selecionado
	filter   string // filtro ativo
	filtered []any   // dados filtrados

	// Estado de detalhe
	selectedItem any // item em drill-down

	// Estado de input (filter/command)
	inputBuffer textinput.Model

	// Estado de UI
	width     int
	height    int
	clock     time.Time
	quitting  bool
	loading   bool
	toast     string
	toastTime time.Time
	errorMsg  string

	// Componentes
	sideList    list.Model      // side panel
	mainTable   table.Model     // main panel (list view)
	detailView  viewport.Model  // main panel (detail view)
	spinner     spinner.Model   // loading state

	// Dados
	services       []ServiceMetrics
	nodes          []NodeInfo
	agents         []AgentInfo
	tasks          []TaskInfo
	alerts         []AlertInfo
	recommendations []RecommendationInfo

	// Buffers de métricas (ring buffer por serviço)
	metricsBuffer map[string]*RingBuffer

	// Ponte SSE → Bubble Tea
	sseCh   chan SSEEvent
	sseDone chan struct{}

	// Tema e skins
	theme *Theme

	// Contexto / cancelamento
	ctx    context.Context
	cancel context.CancelFunc
}
```

### 3.2 TabID e constantes

```go
type TabID int

const (
	TabServices TabID = iota
	TabNodes
	TabAgents
	TabTasks
	TabAlerts
	TabRecommendations
)

var tabNames = []string{
	"[1] Services",
	"[2] Nodes",
	"[3] Agents",
	"[4] Tasks",
	"[5] Alerts",
	"[6] Recs",
}

var tabSSETopics = map[TabID][]string{
	TabServices:        {"services", "metrics"},
	TabNodes:           {"nodes"},
	TabAgents:          {"agents"},
	TabTasks:           {"tasks"},
	TabAlerts:          {"events", "metrics"},
	TabRecommendations: {"services"},
}
```

### 3.3 Init

```go
func InitialModel(ctx context.Context, sseURL string) DashboardModel {
	innerCtx, cancel := context.WithCancel(ctx)

	// Inicializar componentes
	s := spinner.New()
	s.Spinner = spinner.Dot

	ti := textinput.New()
	ti.Prompt = ""

	m := DashboardModel{
		activeTab:      TabServices,
		viewMode:       ViewList,
		focusedPanel:   PanelMain,
		layoutMode:     LayoutLandscape,
		metricsBuffer:  make(map[string]*RingBuffer),
		sseCh:          make(chan SSEEvent, 64),
		sseDone:        make(chan struct{}),
		theme:          LoadTheme(),
		ctx:            innerCtx,
		cancel:         cancel,
		spinner:        s,
		inputBuffer:    ti,
	}

	// Inicia consumidor SSE em goroutine dedicada
	go sseBridge(innerCtx, sseURL, m.sseCh, m.sseDone)

	return m
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		tickEvery(),           // relógio
		waitForSSE(m.sseCh),   // próxima métrica SSE
		m.spinner.Tick,        // spinner
	)
}
```

### 3.4 Update — roteamento de mensagens

```go
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ─── Resize ─────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.calculateLayout()
		return m, nil

	// ─── Tick (relógio) ─────────────────────────────────────
	case tickMsg:
		m.clock = time.Time(msg)
		// Limpar toast após 3 segundos
		if m.toast != "" && time.Since(m.toastTime) > 3*time.Second {
			m.toast = ""
		}
		return m, tea.Batch(tickEvery())

	// ─── SSE Event ──────────────────────────────────────────
	case SSEEvent:
		cmds = append(cmds, m.handleSSE(msg))
		cmds = append(cmds, waitForSSE(m.sseCh))

	// ─── Spinner ────────────────────────────────────────────
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	// ─── Teclado ────────────────────────────────────────────
	case tea.KeyMsg:
		// Teclas globais (sempre ativas, independente do modo)
		switch msg.String() {
		case "q", "ctrl+c":
			if m.viewMode == ViewHelp || m.viewMode == ViewConfirm {
				// q fecha overlay, não sai
				m.viewMode = ViewList
				return m, nil
			}
			m.cancel()
			m.quitting = true
			return m, tea.Quit

		case "?":
			if m.viewMode == ViewHelp {
				m.viewMode = ViewList
			} else {
				m.viewMode = ViewHelp
			}
			return m, nil

		case "esc":
			return m.handleEsc()

		case "1", "2", "3", "4", "5", "6":
			m.activeTab = TabID(int(msg.String()[0] - '1'))
			m.viewMode = ViewList
			m.cursor = 0
			m.filter = ""
			return m, nil

		case "tab":
			m.cycleFocus(1)
			return m, nil

		case "shift+tab":
			m.cycleFocus(-1)
			return m, nil

		case "r":
			return m, m.requestRefresh()
		}

		// Roteamento baseado no modo de visualização
		switch m.viewMode {
		case ViewCommand:
			return m.handleCommandMode(msg, cmds)
		case ViewFilter:
			return m.handleFilterMode(msg, cmds)
		case ViewConfirm:
			return m.handleConfirmMode(msg, cmds)
		case ViewHelp:
			// Help overlay só responde a q/Esc (já tratado acima)
			return m, nil
		case ViewDetail:
			return m.handleDetailMode(msg, cmds)
		case ViewList:
			return m.handleListMode(msg, cmds)
		}

	// ─── Mouse ──────────────────────────────────────────────
	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, tea.Batch(cmds...)
}
```

### 3.5 calculateLayout

```go
const (
	MinWidth  = 80
	MinHeight = 24
)

func (m *DashboardModel) calculateLayout() {
	if m.width < MinWidth || m.height < MinHeight {
		return // tooSmall será tratado no View()
	}

	// Detectar portrait mode (lazydocker pattern)
	if m.width <= 84 && m.height > 45 {
		m.layoutMode = LayoutPortrait
	} else {
		m.layoutMode = LayoutLandscape
	}

	// Alturas fixas: header(1) + tabbar(1) + breadcrumb(1) + footer(1) = 4
	// Borders: 2 horizontais (top+bottom do content) = 2
	// Total fixo: 6
	availableHeight := m.height - 6
	availableWidth := m.width - 3 // 1 vertical border + 2 padding

	if m.layoutMode == LayoutPortrait {
		// Vertical: side em cima (33%), main embaixo (67%)
		sideHeight := availableHeight / 3
		mainHeight := availableHeight - sideHeight

		m.sideList.SetWidth(m.width - 4)
		m.sideList.SetHeight(sideHeight - 2) // -2 for borders (Golden Rule #1)

		m.mainTable.SetWidth(m.width - 4)
		m.mainTable.SetHeight(mainHeight - 2)

		m.detailView.Width = m.width - 4
		m.detailView.Height = mainHeight - 2
	} else {
		// Horizontal: side esquerda (33%), main direita (67%)
		sideWidth := availableWidth / 3
		mainWidth := availableWidth - sideWidth

		m.sideList.SetWidth(sideWidth - 2)
		m.sideList.SetHeight(availableHeight - 2)

		m.mainTable.SetWidth(mainWidth - 2)
		m.mainTable.SetHeight(availableHeight - 2)

		m.detailView.Width = mainWidth - 2
		m.detailView.Height = availableHeight - 2
	}
}
```

---

## 4. Navegação — Keybindings Completos

### 4.1 Teclas globais (sempre ativas)

| Tecla | Ação | Descrição |
|-------|------|-----------|
| `q` | Quit | Sai do dashboard (ou fecha overlay se help/confirm ativo) |
| `Ctrl+c` | Force quit | Sai incondicionalmente |
| `?` | Help | Alterna overlay de help contextual |
| `:` | Command mode | Entra em command mode (k9s style) |
| `1`-`6` | Switch tab | Troca para tab específica |
| `Tab` | Next panel | Move foco para próximo painel (side → main) |
| `Shift+Tab` | Prev panel | Move foco para painel anterior (main → side) |
| `r` | Refresh | Força refresh imediato dos dados |
| `Esc` | Back | Volta de drill-down / cancela filter / cancela command |

### 4.2 List View (modo default — tabela/lista com cursor)

| Tecla | Ação | Vim equiv. |
|-------|------|------------|
| `j` ou `↓` | Próximo item | `j` |
| `k` ou `↑` | Item anterior | `k` |
| `g` | Primeiro item (top) | `gg` |
| `G` | Último item (bottom) | `G` |
| `Ctrl+u` | Page up | `Ctrl+u` |
| `Ctrl+d` | Page down | `Ctrl+d` |
| `Enter` | Drill-down (detail view) | `Enter` |
| `/` | Entrar filter mode | `/` |
| `n` | Próximo match de filtro | `n` |
| `N` | Match anterior de filtro | `N` |
| `a` | Apply (recommendations) | — |
| `d` | Delete item (com confirmação) | — |
| `e` | Edit config | — |
| `l` | View logs | — |
| `s` | Shell/exec (services) | — |
| `y` | YAML/describe | — |

### 4.3 Detail View (drill-down)

| Tecla | Ação |
|-------|------|
| `Esc` | Voltar para List View |
| `j` ou `↓` | Scroll down (viewport) |
| `k` ou `↑` | Scroll up (viewport) |
| `g` | Scroll to top |
| `G` | Scroll to bottom |
| `a` | Apply recommendation (se aplicável) |
| `r` | Rollback (se aplicável) |
| `d` | Delete resource (com confirmação) |
| `e` | Edit config |
| `l` | View logs |
| `s` | Shell into container |
| `y` | Copy YAML |

### 4.4 Command Mode (`:`)

| Tecla | Ação |
|-------|------|
| `Enter` | Executar comando |
| `Esc` | Cancelar command mode |
| `Tab` | Autocomplete (comandos/nomes) |

**Comandos suportados:**

| Comando | Ação | Exemplo |
|---------|------|---------|
| `:services` | Vai para tab Services | — |
| `:nodes` | Vai para tab Nodes | — |
| `:agents` | Vai para tab Agents | — |
| `:tasks` | Vai para tab Tasks | — |
| `:alerts` | Vai para tab Alerts | — |
| `:recs` | Vai para tab Recommendations | — |
| `:apply <service>` | Apply recommendation | `:apply api` |
| `:rollback <service>` | Rollback service | `:rollback api` |
| `:filter <pattern>` | Filtro global | `:filter api` |
| `:q` | Quit | — |
| `:help` | Mostra help | — |
| `:refresh` | Refresh dados | — |

### 4.5 Filter Mode (`/`)

| Tecla | Ação |
|-------|------|
| `Enter` | Aplicar filtro |
| `Esc` | Cancelar filtro |
| Regex | Filtro por regex |

**Exemplos de regex:**
- `api` — match exato (contém "api")
- `.*-dev` — serviços terminados em -dev
- `node-[12]` — node-1 ou node-2
- `!api` — inverso (tudo que não contém "api")

### 4.6 Confirmation Modal

| Tecla | Ação |
|-------|------|
| `y` | Confirmar ação |
| `n` | Cancelar |
| `Esc` | Cancelar |

### 4.7 HotKeys customizáveis

Definidas em `~/.resma/hotkeys.yaml` (estilo k9s):

```yaml
hotKeys:
  shift-1:
    shortCut: Shift-1
    description: Go to Services
    command: services
  shift-2:
    shortCut: Shift-2
    description: Go to Alerts
    command: alerts
  shift-3:
    shortCut: Shift-3
    description: Apply recommendation for api
    command: apply api
  ctrl-r:
    shortCut: Ctrl-R
    description: Force refresh
    command: refresh
```

HotKeys aparecem no help overlay (`?`) e são recarregadas automaticamente (file watch).

### 4.8 Mouse support

Habilitado via `tea.WithMouseCellMotion()`:

| Event | Ação |
|-------|------|
| Click em item da lista | Seleciona item |
| Click em tab | Troca para tab |
| Double-click em item | Drill-down (Enter) |
| Scroll up/down | Scroll lista/tabela/viewport |
| Click no footer | — (futuro: clicar em keybinding) |

Mouse é **opcional** — habilitado por config (`~/.resma/config.yaml` → `tui.mouse: true`).
Default: `false` (seguindo k9s que tem `enableMouse: false` por padrão).

---

## 5. Drill-Down — Detail Views

### 5.1 Service Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ Service: api                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Status: running   Replicas: 3/3                                 │
│                                                                 │
│ Resources:                                                      │
│   CPU:  45.2% (p95: 78%)   Limit: 2 cores                      │
│   MEM:  62.1% (p95: 85%)   Limit: 4Gi                          │
│   OOMs (7d): 0                                                  │
│                                                                 │
│ CPU Trend (last 60s):                                           │
│   ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆▅▄▃▂▁                             │
│                                                                 │
│ Containers:                                                     │
│   task-1  node-2  running  2h15m  CPU: 42%  MEM: 60%          │
│   task-2  node-3  running  2h15m  CPU: 48%  MEM: 65%          │
│   task-3  node-4  running  2h15m  CPU: 46%  MEM: 61%          │
│                                                                 │
│ Recommendations:                                                │
│   [a] Apply: CPU 2 cores, MEM 4Gi (balanced, low risk)        │
│   Reason: CPU p95 at 78%. Current limits adequate.            │
│                                                                 │
│ Actions: [e] edit  [d] delete  [l] logs  [s] shell  [y] yaml  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Node Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ Node: node-2                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Hostname: swarm-worker-01   Role: worker   Status: ready       │
│                                                                 │
│ Resources:                                                      │
│   CPU:  78.0%   Memory: 72.0%   Disk: 62.0%                   │
│                                                                 │
│ Agent:                                                          │
│   Status: active   Version: 7.8.1   Last seen: 1s ago         │
│   Services monitored: 6                                        │
│                                                                 │
│ Services running on this node:                                  │
│   api (task-1)           running  2h15m                        │
│   worker-3 (task-5)      running  1h45m                        │
│   batch-processor (task-7)  failed  0                          │
│   redis-cache (task-10)  running  12h00m                      │
│                                                                 │
│ Actions: [d] drain  [l] agent logs  [r] restart agent          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.3 Alert Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ Alert: CRITICAL — ml-inference                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Level: critical   Service: ml-inference   Time: 14:32:01      │
│                                                                 │
│ Message:                                                        │
│   Memory usage 89% — approaching OOM threshold (90%)          │
│                                                                 │
│ Context:                                                        │
│   Current memory: 8.0Gi / 9.0Gi (89%)                         │
│   Memory limit: 9.0Gi                                          │
│   OOM events (last 24h): 0                                     │
│                                                                 │
│ Related events:                                                 │
│   14:30:00  Memory spike detected (75% → 89%)                │
│   14:28:00  CPU spike detected (8% → 12%)                    │
│                                                                 │
│ Recommendations:                                                │
│   [a] Apply: MEM 8Gi → 12Gi (conservative, high risk)        │
│   Reason: Memory at 89%, OOM risk. Increase mem limit.        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.4 Recommendation Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ Recommendation: ml-inference                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Risk level: high   Tier: conservative                          │
│                                                                 │
│ Current limits:                                                 │
│   CPU: 4 cores   Memory: 8Gi                                  │
│                                                                 │
│ Recommended limits:                                             │
│   CPU: 4 cores   Memory: 12Gi                                 │
│                                                                 │
│ Reason:                                                        │
│   Memory at 89%, OOM risk. Increase mem limit to 12Gi.        │
│                                                                 │
│ Before/After (estimated):                                      │
│   Memory usage: 89% → 74%                                      │
│   OOM risk:     high → low                                     │
│                                                                 │
│ Actions: [a] apply  [r] rollback after apply  [d] dismiss     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.5 Agent Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ Agent: node-4                                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Node: node-4   Hostname: swarm-worker-03                       │
│ Status: active   Version: 7.8.0   Last seen: 5s ago           │
│ Services monitored: 7                                          │
│                                                                 │
│ Health:                                                         │
│   Heartbeat: OK (5s ago)                                       │
│   Metrics push: OK (3s ago)                                    │
│   Buffer: 0/1000 (0%)                                          │
│                                                                 │
│ Services on this node:                                          │
│   api, ml-inference, worker-3, postgres, redis-cache, ...     │
│                                                                 │
│ Actions: [r] restart agent  [l] agent logs  [d] drain node    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 5.6 Task Detail

```
┌─────────────────────────────────────────────────────────────────┐
│ Task: task-6                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Service: worker-3   Node: node-3                               │
│ Status: failed   Desired: running                              │
│ Uptime: 0   Restarts: 3                                        │
│                                                                 │
│ History (last 5 restarts):                                     │
│   14:25:03  failed — exit code 137 (OOM killed)              │
│   14:20:15  started                                            │
│   14:15:02  failed — exit code 137 (OOM killed)              │
│   14:10:00  started                                            │
│   14:05:00  failed — exit code 1                              │
│                                                                 │
│ Actions: [l] logs  [r] restart task  [d] remove task          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Componentes

### 6.1 Componentes Bubble Tea utilizados

| Componente | Uso | Features usadas |
|------------|-----|-----------------|
| `bubbles/table` | Main panel (list view) | Focus/Blur, cursor, KeyMap (j/k, PageUp/Down, GotoTop/Bottom), styles (header, selected) |
| `bubbles/list` | Side panel | Filterable, delegate, selected style |
| `bubbles/viewport` | Main panel (detail view) | Scroll vertical, SoftWrap, FillHeight, MouseWheelEnabled |
| `bubbles/textinput` | Command mode (`:`) e filter (`/`) | Prompt customizável, cursor |
| `bubbles/spinner` | Loading states | Dot spinner, Tick |
| `bubbles/key` | Keybindings | NewBinding, WithKeys, WithHelp |

### 6.2 Header

```go
func renderHeader(m DashboardModel) string {
	title := m.theme.TitleStyle.Render("RESMA")
	subtitle := m.theme.SubtitleStyle.Render("  Docker Swarm Resource Manager")
	clock := m.theme.ClockStyle.Render(m.clock.Format("2006-01-02 15:04:05"))

	status := m.theme.OnlineStyle.Render("● Online")
	if m.loading {
		status = m.theme.SpinnerStyle.Render(m.spinner.View() + " Loading...")
	} else if m.errorMsg != "" {
		status = m.theme.ErrorStyle.Render("✗ " + m.errorMsg)
	}

	padding := strings.Repeat(" ", max(0, m.width-60))
	return lipgloss.JoinHorizontal(lipgloss.Top, title, subtitle, padding, clock, "  ", status)
}
```

### 6.3 Tab Bar

```go
func renderTabBar(m DashboardModel) string {
	var tabs string
	for i, name := range tabNames {
		if TabID(i) == m.activeTab {
			tabs += m.theme.TabActiveStyle.Render(name)
		} else {
			tabs += m.theme.TabInactiveStyle.Render(name)
		}
	}
	return m.theme.TabBarStyle.Render(tabs)
}
```

### 6.4 ServiceTable (bubbles/table)

```go
func newServiceTable(theme *Theme) table.Model {
	columns := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "REPLICAS", Width: 10},
		{Title: "CPU%", Width: 8},
		{Title: "MEM%", Width: 8},
		{Title: "STATUS", Width: 12},
		{Title: "TREND", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// KeyMap estilo vim
	t.KeyMap = table.KeyMap{
		LineUp:       key.NewBinding(key.WithKeys("k", "up")),
		LineDown:     key.NewBinding(key.WithKeys("j", "down")),
		PageUp:       key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
		PageDown:     key.NewBinding(key.WithKeys("ctrl+d", "pgdown")),
		GotoTop:      key.NewBinding(key.WithKeys("g", "home")),
		GotoBottom:   key.NewBinding(key.WithKeys("G", "end")),
	}

	// Styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.BorderColor).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.AccentColor)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(theme.PrimaryColor).
		Bold(true)
	t.SetStyles(s)

	return t
}
```

### 6.5 SideList (bubbles/list)

```go
func newSideList(items []list.Item, theme *Theme) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(theme.PrimaryColor).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#FFFFFF"))

	l := list.New(items, delegate, 0, 0)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)

	return l
}
```

### 6.6 DetailView (bubbles/viewport)

```go
func newDetailView() viewport.Model {
	v := viewport.New(0, 0)
	v.MouseWheelEnabled = true
	v.SoftWrap = true
	v.FillHeight = true
	return v
}
```

### 6.7 Sparkline (asciigraph)

```go
func renderSparkline(points []float64, width int) string {
	if len(points) == 0 {
		return strings.Repeat("▁", width)
	}
	return asciigraph.Plot(points,
		asciigraph.Height(1),
		asciigraph.Width(width),
		asciigraph.SeriesColors(asciigraph.Default),
	)
}
```

### 6.8 LiveChart (asciigraph multi-series)

```go
func renderLiveChart(buf *RingBuffer, theme *Theme, width, height int) string {
	cpuPoints := buf.Slice(SeriesCPU, 60)
	memPoints := buf.Slice(SeriesMem, 60)
	chart := asciigraph.PlotMany(
		[][]float64{cpuPoints, memPoints},
		asciigraph.Width(width-4),
		asciigraph.Height(height-4),
		asciigraph.SeriesColors(asciigraph.Blue, asciigraph.Magenta),
		asciigraph.Caption("CPU (azul) / MEM (magenta) — últimos 60s"),
	)
	return theme.ChartBorder.Render(chart)
}
```

### 6.9 MetricCard

```
┌─ CPU ───────────────────┐
│  78.3%        ↑ +12.1   │
│  ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁       │
└──────────────────────────┘
```

```go
func renderMetricCard(title, value, delta string, points []float64, theme *Theme) string {
	header := theme.CardTitleStyle.Render(title)
	val := theme.CardValueStyle.Render(value)
	d := theme.DeltaUpStyle.Render(delta)
	spark := renderSparkline(points, 20)
	body := lipgloss.JoinVertical(0, val, "  "+d, spark)
	return theme.CardBorder.Render(lipgloss.JoinVertical(0, header, body))
}
```

### 6.10 Breadcrumb

```go
func renderBreadcrumb(m DashboardModel) string {
	parts := []string{tabNames[m.activeTab][4:]} // remove "[N] " prefix
	if m.viewMode == ViewDetail && m.selectedItem != nil {
		parts = append(parts, getItemName(m.selectedItem))
	}
	return m.theme.BreadcrumbStyle.Render(strings.Join(parts, " > "))
}
```

### 6.11 Footer contextual

```go
func renderFooter(m DashboardModel) string {
	var items []string
	switch m.viewMode {
	case ViewList:
		items = []string{
			"q quit", "j/k move", "Enter detail", "/ filter",
			": command", "? help",
		}
		// Adicionar action keys específicas da tab
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
		items = []string{"Enter execute", "Tab autocomplete", "Esc cancel"}
	case ViewHelp:
		items = []string{"q/Esc close"}
	case ViewConfirm:
		items = []string{"y confirm", "n/Esc cancel"}
	}
	return m.theme.FooterStyle.Render(strings.Join(items, " · "))
}
```

### 6.12 Toast notifications

```go
func renderToast(m DashboardModel) string {
	if m.toast == "" {
		return ""
	}
	return m.theme.ToastStyle.Render(m.toast)
}
```

Toast aparece no topo do content area por 3 segundos, depois desaparece.

### 6.13 Help overlay

```
┌──────────────────────────────────────────────────────────────┐
│  KEYBINDINGS — Services Tab                                  │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Navigation:                                                 │
│    j/k or ↑/↓    Move cursor                                 │
│    g/G            Go to top/bottom                           │
│    Ctrl+u/d       Page up/down                               │
│    Enter          Drill-down (detail view)                  │
│    Esc            Back to list view                          │
│                                                              │
│  Actions:                                                    │
│    a              Apply recommendation                      │
│    d              Delete service (with confirmation)        │
│    e              Edit config                                │
│    l              View logs                                  │
│    s              Shell/exec                                 │
│    y              YAML/describe                              │
│                                                              │
│  Filter:                                                     │
│    /              Enter filter mode (regex)                 │
│    n/N            Next/previous filter match                │
│                                                              │
│  Global:                                                     │
│    q              Quit                                       │
│    1-6            Switch tab                                 │
│    Tab/Shift+Tab  Switch panel                               │
│    :              Command mode                               │
│    ?              This help                                  │
│    r              Refresh                                    │
│                                                              │
│  HotKeys:                                                    │
│    Shift-1        Go to Services                             │
│    Shift-2        Go to Alerts                               │
│    Shift-3        Apply recommendation for api              │
│                                                              │
│  Press q or Esc to close                                     │
└──────────────────────────────────────────────────────────────┘
```

### 6.14 Confirmation modal

```
┌──────────────────────────────────────────────────────────────┐
│  ⚠ Delete service "api"?                                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  This will stop all 3 replicas and archive the service.      │
│  This action cannot be undone without `resma services restore`.│
│                                                              │
│  [y] yes   [n] no   [Esc] cancel                            │
└──────────────────────────────────────────────────────────────┘
```

---

## 7. View — Renderização

### 7.1 View principal

```go
func (m DashboardModel) View() string {
	if m.quitting {
		return ""
	}

	// Terminal muito pequeno
	if m.width < MinWidth || m.height < MinHeight {
		return m.theme.ErrorStyle.Render(
			fmt.Sprintf("Terminal too small: %dx%d (min: %dx%d)\nPress q to quit",
				m.width, m.height, MinWidth, MinHeight),
		)
	}

	// Help overlay (fullscreen)
	if m.viewMode == ViewHelp {
		return m.renderHelpOverlay()
	}

	// Header + Tab bar
	header := renderHeader(m)
	tabBar := renderTabBar(m)

	// Content (depende do modo)
	var content string
	switch m.viewMode {
	case ViewCommand:
		content = m.renderCommandInput()
	case ViewFilter:
		content = m.renderFilterInput()
	case ViewConfirm:
		content = m.renderConfirmModal()
	default:
		content = m.renderContent()
	}

	// Breadcrumb + Footer
	breadcrumb := renderBreadcrumb(m)
	footer := renderFooter(m)

	// Toast overlay (topo do content)
	toast := renderToast(m)
	if toast != "" {
		content = lipgloss.JoinVertical(lipgloss.Left, toast, content)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabBar,
		content,
		breadcrumb,
		footer,
	)
}
```

### 7.2 renderContent (list vs detail)

```go
func (m DashboardModel) renderContent() string {
	if m.viewMode == ViewDetail {
		return m.renderDetailView()
	}
	return m.renderListView()
}

func (m DashboardModel) renderListView() string {
	// Side panel (lista filtrável)
	sideView := m.sideList.View()

	// Main panel (tabela com cursor + cards)
	mainView := m.renderMainPanel()

	// Aplicar bordas (foco indicado por cor da borda)
	if m.focusedPanel == PanelSide {
		sideView = m.theme.ActiveBorderStyle.Render(sideView)
		mainView = m.theme.InactiveBorderStyle.Render(mainView)
	} else {
		sideView = m.theme.InactiveBorderStyle.Render(sideView)
		mainView = m.theme.ActiveBorderStyle.Render(mainView)
	}

	// Layout: horizontal (landscape) ou vertical (portrait)
	if m.layoutMode == LayoutPortrait {
		return lipgloss.JoinVertical(lipgloss.Left, sideView, mainView)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sideView, mainView)
}

func (m DashboardModel) renderDetailView() string {
	// Detail view ocupa todo o main panel com viewport scrollable
	detail := m.detailView.View()
	return m.theme.ActiveBorderStyle.Render(detail)
}
```

### 7.3 renderMainPanel (por tab)

```go
func (m DashboardModel) renderMainPanel() string {
	switch m.activeTab {
	case TabServices:
		return m.renderServicesPanel()
	case TabNodes:
		return m.renderNodesPanel()
	case TabAgents:
		return m.renderAgentsPanel()
	case TabTasks:
		return m.renderTasksPanel()
	case TabAlerts:
		return m.renderAlertsPanel()
	case TabRecommendations:
		return m.renderRecommendationsPanel()
	}
	return ""
}
```

Cada tab renderiza sua `bubbles/table` específica com colunas apropriadas + cards de métricas + sparklines.

---

## 8. Skins / Themes

### 8.1 Skin file YAML (estilo k9s)

Skins ficam em `~/.resma/skins/` e são arquivos YAML:

```yaml
# ~/.resma/skins/dark.yaml
resma:
  body:
    fgColor: "#e0def4"
    bgColor: "#191724"
    logoColor: "#7D56F3"

  header:
    fgColor: "#ffffff"
    bgColor: "#7D56F3"
    clockColor: "#6B7280"
    statusColor: "#04E762"

  tabs:
    activeColor: "#7D56F3"
    inactiveColor: "#6B7280"
    activeFg: "#ffffff"
    inactiveFg: "#6B7280"

  table:
    headerFg: "#00D9FF"
    headerBorder: "#3D3D5C"
    selectedFg: "#ffffff"
    selectedBg: "#7D56F3"
    borderColor: "#3D3D5C"

  borders:
    activeColor: "#7D56F3"
    inactiveColor: "#3D3D5C"

  status:
    success: "#04E762"
    warning: "#FFB400"
    error: "#FF5C5C"
    muted: "#6B7280"
    accent: "#00D9FF"

  sparkline:
    color: "#00D9FF"

  chart:
    cpuColor: "#00D9FF"
    memColor: "#FF00FF"
    border: "#3D3D5C"

  toast:
    successBg: "#04E762"
    successFg: "#000000"
    errorBg: "#FF5C5C"
    errorFg: "#ffffff"

  breadcrumb:
    fgColor: "#6B7280"
    separator: ">"
```

### 8.2 Skin selection

| Prioridade | Fonte | Exemplo |
|-----------|-------|---------|
| 1 (alta) | Env var | `RESMA_SKIN=dark` |
| 2 | Config file | `tui.skin: dark` em `~/.resma/config.yaml` |
| 3 (baixa) | Auto-detect | termenv detecta dark/light terminal |

### 8.3 Skins incluídas (built-in)

| Skin | Descrição |
|------|-----------|
| `dark` | Tema escuro padrão (roxo/cyan) |
| `light` | Tema claro para terminais com fundo branco |
| `nord` | Paleta Nord (azul/gelo) |
| `dracula` | Paleta Dracula (roxo/rosa) |
| `transparent` | Preserva background do terminal |

### 8.4 Theme struct (Go)

```go
type Theme struct {
	// Cores base
	PrimaryColor   lipgloss.Color
	AccentColor    lipgloss.Color
	SuccessColor   lipgloss.Color
	WarningColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	MutedColor     lipgloss.Color
	BorderColor    lipgloss.Color
	BackgroundColor lipgloss.Color
	ForegroundColor lipgloss.Color

	// Estilos
	TitleStyle       lipgloss.Style
	SubtitleStyle    lipgloss.Style
	TabActiveStyle   lipgloss.Style
	TabInactiveStyle lipgloss.Style
	TabBarStyle      lipgloss.Style
	HeaderStyle      lipgloss.Style
	FooterStyle      lipgloss.Style
	BreadcrumbStyle  lipgloss.Style
	ContentStyle     lipgloss.Style

	// Bordas
	ActiveBorderStyle   lipgloss.Style
	InactiveBorderStyle lipgloss.Style

	// Tabela
	TableHeaderStyle lipgloss.Style
	TableSelectedStyle lipgloss.Style

	// Cards
	CardBorder      lipgloss.Style
	CardTitleStyle  lipgloss.Style
	CardValueStyle  lipgloss.Style

	// Status
	OnlineStyle  lipgloss.Style
	SpinnerStyle lipgloss.Style
	ErrorStyle   lipgloss.Style
	SuccessStyle lipgloss.Style
	WarningStyle lipgloss.Style

	// Chart
	ChartBorder lipgloss.Style

	// Toast
	ToastStyle lipgloss.Style

	// Clock
	ClockStyle lipgloss.Style
}

func LoadTheme() *Theme {
	// 1. Detectar skin via env/config
	// 2. Carregar YAML se existir
	// 3. Fallback para dark default
	// 4. Aplicar colorprofile.Detect() para TrueColor/256/ANSI
	return loadSkin("dark")
}
```

---

## 9. Configuração TUI

### 9.1 Config file (`~/.resma/config.yaml`)

```yaml
tui:
  # Skin/theme
  skin: dark              # dark | light | nord | dracula | transparent | custom-name

  # Mouse
  mouse: false            # habilita tea.WithMouseCellMotion()

  # Refresh rate (segundos entre polls SSE)
  refreshRate: 2

  # Layout
  sidePanelWidth: 0.33    # proporção do side panel (0.0-1.0)
  expandFocusedPanel: false  # accordion effect (lazydocker)
  border: normal          # normal | rounded | double | hidden

  # Display
  showBreadcrumb: true
  showFooter: true
  showSparklines: true
  showLiveChart: true     # gráfico CPU/MEM em tempo real no detail
  sparklinePoints: 20     # número de pontos no sparkline
  liveChartSeconds: 60    # janela do live chart em segundos

  # Behavior
  confirmOnQuit: false    # pede confirmação ao sair
  scrollPastBottom: true  # permite scroll além do último item
  wrapMainPanel: false    # wrap de texto no main panel
```

### 9.2 HotKeys file (`~/.resma/hotkeys.yaml`)

```yaml
hotKeys:
  shift-1:
    shortCut: Shift-1
    description: Go to Services
    command: services
  shift-2:
    shortCut: Shift-2
    description: Go to Alerts
    command: alerts
  shift-3:
    shortCut: Shift-3
    description: Apply recommendation for api
    command: "apply api"
  ctrl-r:
    shortCut: Ctrl-R
    description: Force refresh
    command: refresh
```

HotKeys são recarregadas automaticamente quando o arquivo muda (file watch via fsnotify).

---

## 10. SSE Integration

### 10.1 Tópicos SSE por tab

| Tab | Tópicos SSE consumidos |
|-----|----------------------|
| Services | `services`, `metrics` |
| Nodes | `nodes` |
| Agents | `agents` |
| Tasks | `tasks` |
| Alerts | `events`, `metrics` |
| Recommendations | `services` |

### 10.2 Ponte SSE → Bubble Tea

```go
type SSEEvent struct {
	Topic string
	Data  json.RawMessage
}

func sseBridge(ctx context.Context, sseURL string, ch chan<- SSEEvent, done chan<- struct{}) {
	defer close(done)

	req, _ := http.NewRequest("GET", sseURL, nil)
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			ch <- SSEEvent{Topic: currentEvent, Data: json.RawMessage(data)}
		}
	}
}

func waitForSSE(ch <-chan SSEEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return event
	}
}
```

### 10.3 handleSSE

```go
func (m *DashboardModel) handleSSE(event SSEEvent) tea.Cmd {
	switch event.Topic {
	case "metrics":
		var metrics MetricsPayload
		if err := json.Unmarshal(event.Data, &metrics); err == nil {
			m.updateMetrics(metrics)
		}
	case "services":
		var services []ServiceMetrics
		if err := json.Unmarshal(event.Data, &services); err == nil {
			m.services = services
		}
	case "nodes":
		var nodes []NodeInfo
		if err := json.Unmarshal(event.Data, &nodes); err == nil {
			m.nodes = nodes
		}
	case "agents":
		var agents []AgentInfo
		if err := json.Unmarshal(event.Data, &agents); err == nil {
			m.agents = agents
		}
	case "tasks":
		var tasks []TaskInfo
		if err := json.Unmarshal(event.Data, &tasks); err == nil {
			m.tasks = tasks
		}
	case "events":
		var evt DockerEvent
		if err := json.Unmarshal(event.Data, &evt); err == nil {
			m.handleDockerEvent(evt)
		}
	}
	return nil
}
```

---

## 11. Ring Buffer de Métricas

Cada serviço tem um ring buffer para manter os últimos N pontos de CPU/memória para sparklines e live charts.

```go
const (
	RingBufferSize = 120 // 120 pontos = 2 min a 1s/tick ou 4 min a 2s/tick
)

type RingBuffer struct {
	cpu  []float64
	mem  []float64
	head int
	size int
}

func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		cpu: make([]float64, RingBufferSize),
		mem: make([]float64, RingBufferSize),
	}
}

func (rb *RingBuffer) Push(cpu, mem float64) {
	rb.cpu[rb.head] = cpu
	rb.mem[rb.head] = mem
	rb.head = (rb.head + 1) % RingBufferSize
	if rb.size < RingBufferSize {
		rb.size++
	}
}

func (rb *RingBuffer) Slice(series int, n int) []float64 {
	if n > rb.size {
		n = rb.size
	}
	result := make([]float64, n)
	start := (rb.head - n + RingBufferSize) % RingBufferSize
	for i := 0; i < n; i++ {
		idx := (start + i) % RingBufferSize
		if series == SeriesCPU {
			result[i] = rb.cpu[idx]
		} else {
			result[i] = rb.mem[idx]
		}
	}
	return result
}
```

---

## 12. Focus Management

### 12.1 cycleFocus

```go
func (m *DashboardModel) cycleFocus(direction int) {
	panels := []PanelID{PanelSide, PanelMain}
	currentIdx := 0
	for i, p := range panels {
		if p == m.focusedPanel {
			currentIdx = i
			break
		}
	}

	newIdx := (currentIdx + direction + len(panels)) % len(panels)

	// Blur current
	if m.focusedPanel == PanelMain {
		m.mainTable.Blur()
	}

	// Focus new
	m.focusedPanel = panels[newIdx]
	if m.focusedPanel == PanelMain {
		m.mainTable.Focus()
	}
}
```

### 12.2 Indicação visual de foco

- **Painel focado**: borda roxa (`#7D56F3`) — `ActiveBorderStyle`
- **Painel sem foco**: borda cinza (`#3D3D5C`) — `InactiveBorderStyle`
- **Item selecionado na tabela**: background roxo, texto branco — `TableSelectedStyle`

---

## 13. Command Mode — Implementação

### 13.1 Entrada e saída

```go
func (m *DashboardModel) enterCommandMode() {
	m.viewMode = ViewCommand
	m.inputBuffer.Focus()
	m.inputBuffer.Prompt = ":"
	m.inputBuffer.SetValue("")
}

func (m *DashboardModel) exitCommandMode() {
	m.viewMode = ViewList
	m.inputBuffer.Blur()
}

func (m DashboardModel) handleCommandMode(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := m.executeCommand(m.inputBuffer.Value())
		m.exitCommandMode()
		return m, tea.Batch(append(cmds, cmd)...)
	case "esc":
		m.exitCommandMode()
		return m, nil
	case "tab":
		// Autocomplete
		completed := m.autocomplete(m.inputBuffer.Value())
		m.inputBuffer.SetValue(completed)
		m.inputBuffer.CursorEnd()
		return m, nil
	}

	var cmd tea.Cmd
	m.inputBuffer, cmd = m.inputBuffer.Update(msg)
	return m, tea.Batch(append(cmds, cmd)...)
}
```

### 13.2 executeCommand

```go
func (m *DashboardModel) executeCommand(input string) tea.Cmd {
	input = strings.TrimSpace(strings.TrimPrefix(input, ":"))
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "services":
		m.activeTab = TabServices
	case "nodes":
		m.activeTab = TabNodes
	case "agents":
		m.activeTab = TabAgents
	case "tasks":
		m.activeTab = TabTasks
	case "alerts":
		m.activeTab = TabAlerts
	case "recs", "recommendations":
		m.activeTab = TabRecommendations
	case "apply":
		if len(parts) >= 2 {
			return m.applyRecommendation(parts[1])
		}
	case "rollback":
		if len(parts) >= 2 {
			return m.rollbackService(parts[1])
		}
	case "filter":
		if len(parts) >= 2 {
			m.filter = parts[1]
			m.applyFilter()
		}
	case "q", "quit":
		m.cancel()
		m.quitting = true
		return tea.Quit
	case "help":
		m.viewMode = ViewHelp
	case "refresh":
		return m.requestRefresh()
	}

	m.viewMode = ViewList
	return nil
}
```

### 13.3 Autocomplete

```go
var commandSuggestions = []string{
	"services", "nodes", "agents", "tasks", "alerts", "recs",
	"apply", "rollback", "filter", "quit", "help", "refresh",
}

func (m DashboardModel) autocomplete(input string) string {
	input = strings.TrimPrefix(input, ":")
	for _, cmd := range commandSuggestions {
		if strings.HasPrefix(cmd, input) {
			return cmd
		}
	}
	return input
}
```

---

## 14. Filter Mode — Implementação

```go
func (m *DashboardModel) enterFilterMode() {
	m.viewMode = ViewFilter
	m.inputBuffer.Focus()
	m.inputBuffer.Prompt = "/"
	m.inputBuffer.SetValue(m.filter)
}

func (m *DashboardModel) exitFilterMode() {
	m.viewMode = ViewList
	m.inputBuffer.Blur()
}

func (m DashboardModel) handleFilterMode(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filter = m.inputBuffer.Value()
		m.applyFilter()
		m.exitFilterMode()
		return m, nil
	case "esc":
		m.filter = ""
		m.applyFilter()
		m.exitFilterMode()
		return m, nil
	}

	var cmd tea.Cmd
	m.inputBuffer, cmd = m.inputBuffer.Update(msg)
	return m, tea.Batch(append(cmds, cmd)...)
}

func (m *DashboardModel) applyFilter() {
	if m.filter == "" {
		m.filtered = nil
		return
	}

	// Inverse regex (k9s style: !pattern)
	if strings.HasPrefix(m.filter, "!") {
		pattern := m.filter[1:]
		// Filter OUT items matching pattern
		m.filterItems(pattern, true)
	} else {
		// Filter IN items matching pattern
		m.filterItems(m.filter, false)
	}
}
```

---

## 15. Estrutura de Arquivos — TUI

```
app/cli/internal/tui/
├── dashboard.go              # DashboardModel, Init/Update/View, calculateLayout
├── styles.go                 # Theme struct, LoadTheme, skin loading
├── skins.go                  # YAML skin parser, built-in skins
├── sse.go                    # SSE bridge, waitForSSE, handleSSE
├── ringbuffer.go             # RingBuffer for metrics
├── commands.go               # Command mode (executeCommand, autocomplete)
├── filter.go                 # Filter mode (applyFilter, regex)
├── help.go                   # Help overlay renderer
├── confirm.go                # Confirmation modal
├── toast.go                  # Toast notifications
├── mouse.go                  # Mouse event handler
├── hotkeys.go                # HotKeys loader (YAML)
├── tabs/
│   ├── tabs.go               # Tab interface + TabID constants
│   ├── services_tab.go       # Tab [1] Services — table + detail
│   ├── nodes_tab.go          # Tab [2] Nodes — table + detail
│   ├── agents_tab.go         # Tab [3] Agents — list + detail
│   ├── tasks_tab.go          # Tab [4] Tasks — table + detail
│   ├── alerts_tab.go         # Tab [5] Alerts — feed + detail
│   └── recommendations_tab.go# Tab [6] Recommendations — cards + detail
└── components/
    ├── header.go             # Header (título + relógio + status)
    ├── tabbar.go             # Tab bar com 6 tabs
    ├── breadcrumb.go         # Breadcrumb de navegação
    ├── footer.go             # Footer contextual
    ├── servicetable.go       # bubbles/table configurada para services
    ├── sidelist.go           # bubbles/list para side panel
    ├── detailview.go         # bubbles/viewport para detail
    ├── sparkline.go          # asciigraph sparkline
    ├── livechart.go          # asciigraph multi-series live chart
    ├── metriccard.go         # Card de métrica (valor + sparkline + delta)
    ├── agentlist.go          # Lista de agents (bubbles/list)
    ├── tasklist.go           # Lista de tasks (bubbles/list)
    ├── alertfeed.go          # Feed de alertas (bubbles/viewport)
    └── recommendationcard.go # Card de recomendação (tier + risk + delta)
```

---

## 16. Ações Inline

### 16.1 Keybindings por tipo de recurso

| Recurso | Key | Ação | Confirmação? | Comando equivalente |
|---------|-----|------|--------------|---------------------|
| Services | `a` | Apply recommendation | Sim (se high risk) | `resma recommendations apply <svc> --confirm` |
| Services | `r` | Rollback | Sim | `resma rollback-watches rollback <id> --confirm` |
| Services | `d` | Archive service | Sim | `resma services archive <svc> --confirm` |
| Services | `e` | Edit config | Não (abre editor) | — |
| Services | `l` | View logs | Não | `resma stream service-detail/<svc>` |
| Services | `s` | Shell into container | Não | `docker exec -it <container> sh` |
| Services | `y` | YAML/describe | Não | `resma services inspect <svc>` |
| Nodes | `d` | Drain node | Sim | `docker node update --availability drain <node>` |
| Nodes | `l` | Agent logs | Não | `resma stream agents` |
| Nodes | `r` | Restart agent | Sim | — |
| Agents | `r` | Restart agent | Sim | — |
| Tasks | `l` | Task logs | Não | `docker service logs <task>` |
| Tasks | `r` | Restart task | Sim | — |
| Alerts | `a` | Apply recommendation (se houver) | Contextual | — |
| Recommendations | `a` | Apply | Sim (se high risk) | `resma recommendations apply <svc> --confirm` |
| Recommendations | `d` | Dismiss | Não | — |

### 16.2 Fluxo de confirmação

```
1. Usuário pressiona `d` em service "api"
2. Confirmation modal aparece:
   ┌──────────────────────────────────────────────┐
   │  ⚠ Archive service "api"?                   │
   ├──────────────────────────────────────────────┤
   │  This will stop all 3 replicas.              │
   │  Use `resma services restore api` to undo.  │
   │                                              │
   │  [y] yes   [n] no   [Esc] cancel            │
   └──────────────────────────────────────────────┘
3. Usuário pressiona `y`
4. Ação executada (chama API)
5. Toast aparece: "✓ Service api archived"
6. Lista atualizada via SSE
```

---

## 17. Responsividade e Edge Cases

### 17.1 Terminal resize

```go
case tea.WindowSizeMsg:
	m.width, m.height = msg.Width, msg.Height
	m.calculateLayout()
	return m, nil
```

`calculateLayout()` recalcula todas as dimensões dos componentes a cada resize.

### 17.2 Terminal muito pequeno

```go
if m.width < MinWidth || m.height < MinHeight {
	return m.theme.ErrorStyle.Render(
		fmt.Sprintf("Terminal too small: %dx%d (min: %dx%d)\nPress q to quit",
			m.width, m.height, MinWidth, MinHeight),
	)
}
```

### 17.3 Alt-screen recovery

```go
func main() {
	p := tea.NewProgram(
		tui.InitialModel(ctx, sseURL),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // se mouse habilitado
	)
	if _, err := p.Run(); err != nil {
		// Garantir que terminal seja restaurado mesmo em panic
		fmt.Printf("\x1b[?25h\x1b[?1049l") // show cursor + exit alt screen
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

### 17.4 Unicode width

**Problema:** `fmt.Sprintf("%-20s", name)` usa byte length, não display width. Caracteres Unicode (sparkline `▁▂▃▄▅▆▇█`, emojis) e ANSI codes do `lipgloss.Render()` inflam a contagem.

**Solução:** Usar `bubbles/table` (lida com width internamente) ou `lipgloss.Width()`:

```go
// ❌ ERRADO
fmt.Sprintf("%-20s %8s", name, cpu)

// ✅ CORRETO (bubbles/table)
table.WithColumns([]table.Column{
	{Title: "NAME", Width: 20},
	{Title: "CPU%", Width: 8},
})

// ✅ CORRETO (lipgloss)
lipgloss.NewStyle().Width(20).Render(name)
```

### 17.5 Color bleeding

**Problema:** ANSI codes não resetados "vazam" para linhas seguintes.

**Solução:** `bubbles/table` gerencia reset automaticamente. Para renderização manual, sempre usar `lipgloss.Style.Render()` que insere reset no final.

---

## 18. Roadmap de Implementação do TUI

| Fase | Features | Estimativa |
|------|----------|------------|
| **Fase 1** | Layout two-column, bordas, navegação j/k + Tab, drill-down Enter/Esc, filter `/`, help `?`, footer contextual | — |
| **Fase 2** | Command mode `:`, autocomplete, HotKeys, skins YAML, mouse support | — |
| **Fase 3** | Live chart em tempo real, ring buffer, toast notifications, confirmation modals | — |
| **Fase 4** | Portrait mode, accordion, scroll past bottom, file watch para hotkeys/skins | — |

> **Nota:** Este é o roadmap de implementação do TUI dentro do `resma monitor`.
> Todas as features são parte da spec completa do CLI — nenhuma é cortada.
