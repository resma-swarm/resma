package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode representa o modo de visualização atual.
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
	ViewFilter
	ViewCommand
	ViewHelp
	ViewLogs
	ViewLogDetail
)

// PanelID identifica qual painel está focado.
type PanelID int

const (
	PanelSide PanelID = iota
	PanelMain
)

// TabID identifica uma aba do dashboard.
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

// model é o estado raiz do wireframe.
type model struct {
	activeTab      TabID
	viewMode       ViewMode
	focusedPanel   PanelID
	cursor         int
	width          int
	height         int
	clock          time.Time
	quitting       bool
	inputBuf       string
	filter         string
	selectedItem   string // nome do item em drill-down
	flash          flashMessage
	splash         bool   // mostrar splash no startup
	logCursor      int    // cursor de navegação nos logs
	logFollow      bool   // auto-scroll para o fim (tail)
	logFilter      string // filtro de logs
	filterFromLogs bool   // filter foi aberto da view de logs
	// Sort state para a tabela da view ativa
	sortCol  int       // coluna de ordenação (-1 = nenhuma)
	sortDir  SortDir   // direção da ordenação
	selCol   int       // coluna selecionada pelo Shift+←/→
	selColAt time.Time // timestamp do último Shift+←/→ (para expirar seleção)
}

func initialModel() model {
	return model{
		activeTab:    TabServices,
		viewMode:     ViewList,
		focusedPanel: PanelMain,
		clock:        time.Now(),
		splash:       true,
		logFollow:    true,
		sortCol:      -1,
		selCol:       -1,
		flash:        flashText("Welcome to RESMA Monitor — press ? for help", FlashInfo),
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// selColExpireMsg é enviado após 1s sem Shift+←/→ para limpar a seleção.
type selColExpireMsg time.Time

func selColExpire() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return selColExpireMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.clock = time.Time(msg)
		// Desligar splash após primeiro tick
		if m.splash {
			m.splash = false
		}
		return m, tick()

	case selColExpireMsg:
		// Se a seleção de coluna ainda está ativa e passou >1s,
		// limpar a seleção (usuário soltou o Shift sem pressionar outra tecla)
		if m.selCol >= 0 && time.Since(m.selColAt) > 1*time.Second {
			m.selCol = -1
		}
		// Se selCol ainda >= 0, continuar o timer para checar novamente
		if m.selCol >= 0 {
			return m, selColExpire()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) View() string {
	return renderDashboard(m)
}

// selColLeft move a seleção de coluna para a esquerda (loop infinito).
func (m *model) selColLeft() {
	n := numColsForTab(m.activeTab)
	if n == 0 {
		return
	}
	if m.selCol < 0 {
		m.selCol = 0
	} else {
		m.selCol = (m.selCol - 1 + n) % n
	}
}

// selColRight move a seleção de coluna para a direita (loop infinito).
func (m *model) selColRight() {
	n := numColsForTab(m.activeTab)
	if n == 0 {
		return
	}
	if m.selCol < 0 {
		m.selCol = 0
	} else {
		m.selCol = (m.selCol + 1) % n
	}
}

// numColsForTab retorna o número de colunas da tab ativa.
func numColsForTab(tab TabID) int {
	switch tab {
	case TabServices:
		return 6
	case TabNodes:
		return 7
	case TabAgents:
		return 5
	case TabTasks:
		return 6
	case TabAlerts:
		return 4
	case TabRecommendations:
		return 6
	}
	return 0
}
