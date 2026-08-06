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
}

func initialModel() model {
	return model{
		activeTab:    TabServices,
		viewMode:     ViewList,
		focusedPanel: PanelMain,
		clock:        time.Now(),
		splash:       true,
		logFollow:    true,
		flash:        flashText("Welcome to RESMA Monitor — press ? for help", FlashInfo),
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
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

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) View() string {
	return renderDashboard(m)
}
