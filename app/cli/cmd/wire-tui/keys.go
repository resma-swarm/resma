package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Teclas globais
	switch msg.String() {
	case "q", "ctrl+c":
		if m.viewMode == ViewHelp || m.viewMode == ViewFilter || m.viewMode == ViewCommand {
			m.viewMode = ViewList
			m.inputBuf = ""
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "esc":
		if m.viewMode == ViewLogDetail {
			m.viewMode = ViewLogs
			m.flash = flashText("Back to logs", FlashInfo)
		} else if m.viewMode == ViewDetail {
			m.viewMode = ViewList
			m.selectedItem = ""
			m.flash = flashText("Back to list", FlashInfo)
		} else if m.viewMode == ViewLogs {
			m.viewMode = ViewList
			m.logCursor = 0
			m.logFilter = ""
			m.flash = flashText("Back to list", FlashInfo)
		} else if m.viewMode == ViewFilter || m.viewMode == ViewCommand || m.viewMode == ViewHelp {
			if m.filterFromLogs {
				m.viewMode = ViewLogs
				m.filterFromLogs = false
			} else {
				m.viewMode = ViewList
			}
			m.inputBuf = ""
			m.flash = flashText("", FlashInfo)
		}
		return m, nil

	case "?":
		if m.viewMode == ViewHelp {
			m.viewMode = ViewList
		} else {
			m.viewMode = ViewHelp
		}
		return m, nil

	case ":":
		if m.viewMode == ViewList {
			m.viewMode = ViewCommand
			m.inputBuf = ""
		}
		return m, nil

	case "/":
		if m.viewMode == ViewList {
			m.viewMode = ViewFilter
			m.inputBuf = m.filter
		}
		return m, nil

	case "1":
		return m.switchTab(TabServices)
	case "2":
		return m.switchTab(TabNodes)
	case "3":
		return m.switchTab(TabAgents)
	case "4":
		return m.switchTab(TabTasks)
	case "5":
		return m.switchTab(TabAlerts)
	case "6":
		return m.switchTab(TabRecommendations)

	case "tab":
		if m.viewMode == ViewList {
			if m.focusedPanel == PanelSide {
				m.focusedPanel = PanelMain
			} else {
				m.focusedPanel = PanelSide
			}
		}
		return m, nil

	case "shift+tab":
		if m.viewMode == ViewList {
			if m.focusedPanel == PanelMain {
				m.focusedPanel = PanelSide
			} else {
				m.focusedPanel = PanelMain
			}
		}
		return m, nil

	case "r":
		// mock refresh
		return m, nil
	}

	// Navegação na view de logs
	if m.viewMode == ViewLogs {
		return m.handleLogsKey(msg)
	}

	// Navegação na view de detalhe de log (j/k navega entre entradas)
	if m.viewMode == ViewLogDetail {
		return m.handleLogDetailKey(msg)
	}

	// Navegação na lista (loop infinito como k9s)
	if m.viewMode == ViewList {
		n := m.listLen()
		if n == 0 {
			n = 1
		}
		switch msg.String() {
		case "j", "down":
			m.selCol = -1
			m.cursor = (m.cursor + 1) % n
			return m, nil
		case "k", "up":
			m.selCol = -1
			m.cursor = (m.cursor - 1 + n) % n
			return m, nil
		case "g":
			m.selCol = -1
			m.cursor = 0
			return m, nil
		case "G":
			m.selCol = -1
			m.cursor = n - 1
			return m, nil
		case "l":
			m.selCol = -1
			return m.enterLogs()
		case "enter":
			m.selCol = -1
			return m.enterDetail()
		case "shift+left":
			m.selColLeft()
			m.selColAt = time.Now()
			return m, selColExpire()
		case "shift+right":
			m.selColRight()
			m.selColAt = time.Now()
			return m, selColExpire()
		case "shift+up":
			// Alternar direção reverso: None → Desc → Asc → None
			if m.selCol >= 0 {
				m.sortCol = m.selCol
				m.sortDir = (m.sortDir + 2) % 3
				if m.sortDir == SortNone {
					m.sortCol = -1
				}
			}
			m.selColAt = time.Now()
			return m, selColExpire()
		case "shift+down":
			// Alternar direção: None → Asc → Desc → None
			if m.selCol >= 0 {
				m.sortCol = m.selCol
				m.sortDir = (m.sortDir + 1) % 3
				if m.sortDir == SortNone {
					m.sortCol = -1
				}
			}
			m.selColAt = time.Now()
			return m, selColExpire()
		default:
			// Qualquer outra tecla: desmarcar seleção de coluna
			m.selCol = -1
			return m, nil
		}
	}

	// Input modes
	if m.viewMode == ViewFilter || m.viewMode == ViewCommand {
		return m.handleInput(msg)
	}

	return m, nil
}

func (m model) switchTab(tab TabID) (tea.Model, tea.Cmd) {
	m.activeTab = tab
	m.viewMode = ViewList
	m.cursor = 0
	m.filter = ""
	m.selectedItem = ""
	m.sortCol = -1
	m.sortDir = SortNone
	m.selCol = -1
	m.flash = flashText("Viewing "+tabNames[tab][4:], FlashInfo)
	return m, nil
}

func (m model) enterDetail() (tea.Model, tea.Cmd) {
	items := m.currentItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return m, nil
	}
	m.selectedItem = items[m.cursor]
	m.viewMode = ViewDetail
	m.flash = flashText("Detail: "+m.selectedItem, FlashInfo)
	return m, nil
}

func (m model) enterLogs() (tea.Model, tea.Cmd) {
	items := m.currentItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return m, nil
	}
	m.selectedItem = items[m.cursor]
	m.viewMode = ViewLogs
	m.logCursor = 0
	m.logFollow = true
	m.logFilter = ""
	m.flash = flashText("Logs: "+m.selectedItem, FlashInfo)
	return m, nil
}

func (m model) handleLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	logs := mockLogsFor(m.selectedItem)

	// Aplicar filtro para contar
	if m.logFilter != "" {
		filtered := make([]mockLogEntry, 0)
		for _, l := range logs {
			if strings.Contains(strings.ToLower(l.message), strings.ToLower(m.logFilter)) ||
				strings.Contains(strings.ToLower(l.level), strings.ToLower(m.logFilter)) {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	n := len(logs)
	if n == 0 {
		n = 1
	}

	switch msg.String() {
	case "j", "down":
		m.logFollow = false
		m.logCursor = (m.logCursor + 1) % n
		return m, nil
	case "k", "up":
		m.logFollow = false
		m.logCursor = (m.logCursor - 1 + n) % n
		return m, nil
	case "g":
		m.logFollow = false
		m.logCursor = 0
		return m, nil
	case "G":
		m.logFollow = true
		m.logCursor = n - 1
		return m, nil
	case "f":
		m.logFollow = !m.logFollow
		if m.logFollow {
			m.logCursor = n - 1
		}
		m.flash = flashText("Follow: "+boolStr(m.logFollow), FlashInfo)
		return m, nil
	case " ":
		m.logFollow = !m.logFollow
		if m.logFollow {
			m.logCursor = n - 1
		}
		return m, nil
	case "/":
		m.viewMode = ViewFilter
		m.inputBuf = m.logFilter
		m.filterFromLogs = true
		return m, nil
	case "enter":
		m.viewMode = ViewLogDetail
		m.flash = flashText("Log detail", FlashInfo)
		return m, nil
	}
	return m, nil
}

func boolStr(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// handleLogDetailKey navega entre entradas de log na view de detalhe.
func (m model) handleLogDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	logs := mockLogsFor(m.selectedItem)

	if m.logFilter != "" {
		filtered := make([]mockLogEntry, 0)
		for _, l := range logs {
			if strings.Contains(strings.ToLower(l.message), strings.ToLower(m.logFilter)) ||
				strings.Contains(strings.ToLower(l.level), strings.ToLower(m.logFilter)) {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	n := len(logs)
	if n == 0 {
		n = 1
	}

	switch msg.String() {
	case "j", "down":
		m.logFollow = false
		m.logCursor = (m.logCursor + 1) % n
		return m, nil
	case "k", "up":
		m.logFollow = false
		m.logCursor = (m.logCursor - 1 + n) % n
		return m, nil
	case "g":
		m.logCursor = 0
		return m, nil
	case "G":
		m.logCursor = n - 1
		return m, nil
	}
	return m, nil
}

func (m model) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.viewMode == ViewFilter {
			if m.filterFromLogs {
				m.logFilter = m.inputBuf
				m.viewMode = ViewLogs
				m.filterFromLogs = false
			} else {
				m.filter = m.inputBuf
				m.viewMode = ViewList
			}
		} else if m.viewMode == ViewCommand {
			return m.executeCommand(m.inputBuf)
		}
		m.inputBuf = ""
		return m, nil
	case "backspace":
		if len(m.inputBuf) > 0 {
			m.inputBuf = m.inputBuf[:len(m.inputBuf)-1]
		}
		return m, nil
	default:
		// só aceitar chars imprimíveis
		s := msg.String()
		if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
			m.inputBuf += s
		}
		return m, nil
	}
}

func (m model) executeCommand(input string) (tea.Model, tea.Cmd) {
	m.viewMode = ViewList
	m.inputBuf = ""
	input = strings.TrimSpace(input)
	if input == "" {
		return m, nil
	}
	parts := strings.Fields(input)
	switch parts[0] {
	case "services":
		return m.switchTab(TabServices)
	case "nodes":
		return m.switchTab(TabNodes)
	case "agents":
		return m.switchTab(TabAgents)
	case "tasks":
		return m.switchTab(TabTasks)
	case "alerts":
		return m.switchTab(TabAlerts)
	case "recs", "recommendations":
		return m.switchTab(TabRecommendations)
	case "q", "quit":
		m.quitting = true
		return m, tea.Quit
	case "help":
		m.viewMode = ViewHelp
		m.flash = flashText("Help", FlashInfo)
		return m, nil
	default:
		m.flash = flashText("Unknown command: "+parts[0], FlashErr)
	}
	return m, nil
}
