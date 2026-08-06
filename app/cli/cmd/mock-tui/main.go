// Command mock-tui runs the RESMA CLI dashboard with mock data.
// It demonstrates the 6-tab layout, keybindings, and visual style
// without needing a real API connection.
//
// Usage: go run ./cmd/mock-tui
package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Styles ───────────────────────────────────────────────────────────────────

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7D56F3")
	colorAccent    = lipgloss.Color("#00D9FF")
	colorSuccess   = lipgloss.Color("#04E762")
	colorWarning   = lipgloss.Color("#FFB400")
	colorError     = lipgloss.Color("#FF5C5C")
	colorMuted     = lipgloss.Color("#6B7280")
	colorBorder    = lipgloss.Color("#3D3D5C")
	colorBg        = lipgloss.Color("#1A1A2E")
	colorTabActive = lipgloss.Color("#7D56F3")
	colorTabBg     = lipgloss.Color("#2A2A3E")

	// Style definitions
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 2)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorTabActive).
			Padding(0, 2).
			MarginRight(1)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(colorMuted).
				Background(colorTabBg).
				Padding(0, 2).
				MarginRight(1)

	styleTabBar = lipgloss.NewStyle().
			Padding(0, 1)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	styleContent = lipgloss.NewStyle().
			Padding(1, 2)

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	styleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	styleTableRow = lipgloss.NewStyle()

	styleSuccess   = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	styleWarning   = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	styleError     = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	styleMuted     = lipgloss.NewStyle().Foreground(colorMuted)
	styleHighlight = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			MarginBottom(1)

	styleCardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	styleMetricValue = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSuccess)

	styleSparkline = lipgloss.NewStyle().Foreground(colorAccent)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)
)

// ─── Mock Data ────────────────────────────────────────────────────────────────

type mockService struct {
	name     string
	replicas string
	cpu      float64
	mem      float64
	status   string
	spark    []float64
}

type mockNode struct {
	id       string
	hostname string
	role     string
	cpu      float64
	mem      float64
	disk     float64
	status   string
}

type mockAgent struct {
	nodeID   string
	status   string
	version  string
	lastSeen string
	services int
}

type mockTask struct {
	id      string
	service string
	node    string
	status  string
	desired string
	uptime  string
}

type mockAlert struct {
	level   string
	service string
	message string
	time    string
}

type mockRecommendation struct {
	service string
	tier    string
	cpu     string
	mem     string
	risk    string
	reason  string
}

var mockServices = []mockService{
	{"api", "3/3", 45.2, 62.1, "running", []float64{30, 35, 42, 38, 45, 50, 48, 45, 52, 48, 45}},
	{"ml-inference", "1/1", 12.0, 89.0, "running", []float64{8, 10, 12, 15, 11, 13, 12, 14, 12, 13, 12}},
	{"frontend-dev", "1/1", 3.1, 18.5, "running", []float64{2, 3, 4, 3, 2, 3, 4, 3, 3, 4, 3}},
	{"worker-3", "2/2", 62.0, 40.0, "running", []float64{40, 45, 50, 55, 60, 65, 62, 58, 62, 64, 62}},
	{"postgres", "1/1", 22.0, 71.0, "running", []float64{18, 20, 22, 25, 23, 21, 22, 24, 22, 23, 22}},
	{"redis-cache", "2/2", 8.0, 35.0, "running", []float64{5, 6, 8, 10, 7, 8, 9, 8, 7, 8, 8}},
	{"nginx-proxy", "3/3", 5.0, 12.0, "running", []float64{3, 4, 5, 6, 5, 4, 5, 6, 5, 5, 5}},
	{"batch-processor", "0/2", 0, 0, "stopped", []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
}

var mockNodes = []mockNode{
	{"node-1", "swarm-manager-01", "manager", 52.0, 68.0, 45.0, "ready"},
	{"node-2", "swarm-worker-01", "worker", 78.0, 72.0, 62.0, "ready"},
	{"node-3", "swarm-worker-02", "worker", 35.0, 55.0, 38.0, "ready"},
	{"node-4", "swarm-worker-03", "worker", 91.0, 88.0, 71.0, "ready"},
	{"node-5", "swarm-worker-04", "worker", 0, 0, 0, "down"},
}

var mockAgents = []mockAgent{
	{"node-1", "active", "7.8.1", "2s ago", 8},
	{"node-2", "active", "7.8.1", "1s ago", 6},
	{"node-3", "active", "7.8.1", "3s ago", 5},
	{"node-4", "active", "7.8.0", "5s ago", 7},
	{"node-5", "offline", "7.8.1", "15m ago", 0},
}

var mockTasks = []mockTask{
	{"task-1", "api", "node-2", "running", "running", "2h15m"},
	{"task-2", "api", "node-3", "running", "running", "2h15m"},
	{"task-3", "api", "node-4", "running", "running", "2h15m"},
	{"task-4", "ml-inference", "node-4", "running", "running", "5h30m"},
	{"task-5", "worker-3", "node-2", "running", "running", "1h45m"},
	{"task-6", "worker-3", "node-3", "failed", "running", "0"},
	{"task-7", "batch-processor", "node-2", "failed", "running", "0"},
	{"task-8", "batch-processor", "node-3", "failed", "running", "0"},
	{"task-9", "postgres", "node-1", "running", "running", "12h00m"},
	{"task-10", "redis-cache", "node-2", "running", "running", "12h00m"},
	{"task-11", "redis-cache", "node-3", "running", "running", "12h00m"},
	{"task-12", "nginx-proxy", "node-1", "running", "running", "8h20m"},
}

var mockAlerts = []mockAlert{
	{"critical", "ml-inference", "Memory usage 89% — approaching OOM threshold (90%)", "14:32:01"},
	{"warning", "node-4", "Disk usage 71% — cleanup recommended", "14:28:15"},
	{"warning", "worker-3", "CPU usage 62% — sustained for 15min", "14:25:03"},
	{"critical", "batch-processor", "All replicas stopped (0/2)", "14:10:22"},
	{"info", "node-5", "Agent offline — last seen 15m ago", "14:18:00"},
	{"warning", "postgres", "Memory usage 71% — monitor closely", "14:05:11"},
}

var mockRecommendations = []mockRecommendation{
	{"ml-inference", "conservative", "4 cores", "8Gi", "high", "Memory at 89%, OOM risk. Increase mem limit to 8Gi."},
	{"api", "balanced", "2 cores", "4Gi", "low", "CPU p95 at 78%. Current limits adequate, no change needed."},
	{"worker-3", "aggressive", "1.5 cores", "2Gi", "medium", "CPU sustained 62%. Consider scaling replicas to 3."},
	{"postgres", "conservative", "2 cores", "6Gi", "medium", "Memory at 71%. Increase to 6Gi for headroom."},
	{"batch-processor", "balanced", "1 core", "2Gi", "high", "All replicas failed. Investigate before applying limits."},
	{"redis-cache", "balanced", "0.5 cores", "1Gi", "low", "Stable usage. No changes recommended."},
}

// ─── Sparkline ────────────────────────────────────────────────────────────────

func sparkline(data []float64, width int) string {
	if len(data) == 0 {
		return strings.Repeat(" ", width)
	}
	// Normalize to width
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	result := make([]rune, 0, len(data))
	maxVal := 0.0
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return strings.Repeat(" ", width)
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
	return styleSparkline.Render(string(result))
}

// ─── Tab Definitions ─────────────────────────────────────────────────────────

const (
	tabServices = iota
	tabNodes
	tabAgents
	tabTasks
	tabAlerts
	tabRecommendations
	tabCount
)

var tabNames = []string{
	"[1] Services",
	"[2] Nodes",
	"[3] Agents",
	"[4] Tasks",
	"[5] Alerts",
	"[6] Recs",
}

// ─── Model ────────────────────────────────────────────────────────────────────

type model struct {
	activeTab int
	width     int
	height    int
	clock     time.Time
	quitting  bool
}

func initialModel() model {
	return model{
		activeTab: tabServices,
		clock:     time.Now(),
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
	case tickMsg:
		m.clock = time.Time(msg)
		return m, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "1":
			m.activeTab = tabServices
		case "2":
			m.activeTab = tabNodes
		case "3":
			m.activeTab = tabAgents
		case "4":
			m.activeTab = tabTasks
		case "5":
			m.activeTab = tabAlerts
		case "6":
			m.activeTab = tabRecommendations
		case "tab":
			m.activeTab = (m.activeTab + 1) % tabCount
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
		case "r":
			// Mock refresh — just re-render
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Header
	header := styleHeader.Render("RESMA") +
		styleSubtitle.Render("  Docker Swarm Resource Manager") +
		strings.Repeat(" ", max(0, m.width-60)) +
		styleMuted.Render(m.clock.Format("2006-01-02 15:04:05"))

	// Tab bar
	tabs := styleTabBar.Render("")
	for i, name := range tabNames {
		if i == m.activeTab {
			tabs += styleTabActive.Render(name)
		} else {
			tabs += styleTabInactive.Render(name)
		}
	}

	// Content
	var content string
	switch m.activeTab {
	case tabServices:
		content = renderServices()
	case tabNodes:
		content = renderNodes()
	case tabAgents:
		content = renderAgents()
	case tabTasks:
		content = renderTasks()
	case tabAlerts:
		content = renderAlerts()
	case tabRecommendations:
		content = renderRecommendations()
	}

	// Footer
	footer := styleFooter.Render(
		"q quit  ·  1-6 tabs  ·  Tab next  ·  r refresh  ·  / search  ·  ? help",
	)

	// Assemble
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabs,
		content,
		footer,
	)
}

// ─── Tab Renderers ────────────────────────────────────────────────────────────

func renderServices() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Services — 8 total (7 running, 1 stopped)"))
	sb.WriteString("\n\n")

	// Table header
	sb.WriteString(fmt.Sprintf("%-20s %8s %8s %8s %12s %s\n",
		styleTableHeader.Render("NAME"),
		styleTableHeader.Render("REPLICAS"),
		styleTableHeader.Render("CPU%"),
		styleTableHeader.Render("MEM%"),
		styleTableHeader.Render("STATUS"),
		styleTableHeader.Render("TREND (CPU)"),
	))
	sb.WriteString(styleMuted.Render(strings.Repeat("─", 80)))
	sb.WriteString("\n")

	for _, s := range mockServices {
		cpu := fmt.Sprintf("%.1f", s.cpu)
		mem := fmt.Sprintf("%.1f", s.mem)
		status := s.status
		if s.status == "running" {
			status = styleSuccess.Render("running")
		} else {
			status = styleError.Render("stopped")
		}

		var cpuStyled, memStyled string
		if s.cpu > 70 {
			cpuStyled = styleError.Render(cpu)
		} else if s.cpu > 50 {
			cpuStyled = styleWarning.Render(cpu)
		} else {
			cpuStyled = cpu
		}
		if s.mem > 80 {
			memStyled = styleError.Render(mem)
		} else if s.mem > 60 {
			memStyled = styleWarning.Render(mem)
		} else {
			memStyled = mem
		}

		sb.WriteString(fmt.Sprintf("%-20s %8s %8s %8s %12s  %s\n",
			s.name, s.replicas, cpuStyled, memStyled, status,
			sparkline(s.spark, 20),
		))
	}

	sb.WriteString("\n")
	sb.WriteString(styleMuted.Render("  ▁▂▃▄▅▆▇█  CPU trend (last 11 samples)"))
	sb.WriteString("\n")

	return styleContent.Render(sb.String())
}

func renderNodes() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Nodes — 5 total (4 ready, 1 down)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%-12s %-22s %8s %8s %8s %8s %s\n",
		styleTableHeader.Render("ID"),
		styleTableHeader.Render("HOSTNAME"),
		styleTableHeader.Render("CPU%"),
		styleTableHeader.Render("MEM%"),
		styleTableHeader.Render("DISK%"),
		styleTableHeader.Render("ROLE"),
		styleTableHeader.Render("STATUS"),
	))
	sb.WriteString(styleMuted.Render(strings.Repeat("─", 80)))
	sb.WriteString("\n")

	for _, n := range mockNodes {
		cpu := fmt.Sprintf("%.1f", n.cpu)
		mem := fmt.Sprintf("%.1f", n.mem)
		disk := fmt.Sprintf("%.1f", n.disk)
		status := n.status
		if n.status == "ready" {
			status = styleSuccess.Render("ready")
		} else {
			status = styleError.Render("down")
		}

		var cpuStyled, diskStyled string
		if n.cpu > 80 {
			cpuStyled = styleError.Render(cpu)
		} else if n.cpu > 60 {
			cpuStyled = styleWarning.Render(cpu)
		} else {
			cpuStyled = cpu
		}
		if n.disk > 60 {
			diskStyled = styleWarning.Render(disk)
		} else {
			diskStyled = disk
		}

		sb.WriteString(fmt.Sprintf("%-12s %-22s %8s %8s %8s %8s %s\n",
			n.id, n.hostname, cpuStyled, mem, diskStyled, n.role, status,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(styleCard.Render(
		styleCardTitle.Render("Cluster Summary") + "\n" +
			fmt.Sprintf("  Managers: %s    Workers: %s    Ready: %s    Down: %s",
				styleHighlight.Render("1"), styleHighlight.Render("4"),
				styleSuccess.Render("4"), styleError.Render("1")),
	))

	return styleContent.Render(sb.String())
}

func renderAgents() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Agents — 5 registered (4 active, 1 offline)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%-12s %10s %10s %14s %10s\n",
		styleTableHeader.Render("NODE ID"),
		styleTableHeader.Render("STATUS"),
		styleTableHeader.Render("VERSION"),
		styleTableHeader.Render("LAST SEEN"),
		styleTableHeader.Render("SERVICES"),
	))
	sb.WriteString(styleMuted.Render(strings.Repeat("─", 60)))
	sb.WriteString("\n")

	for _, a := range mockAgents {
		status := a.status
		if a.status == "active" {
			status = styleSuccess.Render("active")
		} else {
			status = styleError.Render("offline")
		}
		sb.WriteString(fmt.Sprintf("%-12s %10s %10s %14s %10d\n",
			a.nodeID, status, a.version, a.lastSeen, a.services,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(styleCard.Render(
		styleCardTitle.Render("Agent Health") + "\n" +
			fmt.Sprintf("  Active: %s/5    Offline: %s/5    Avg latency: %s",
				styleSuccess.Render("4"), styleError.Render("1"),
				styleMuted.Render("1.2s")) + "\n" +
			styleWarning.Render("  ⚠ node-5 offline — last seen 15m ago"),
	))

	return styleContent.Render(sb.String())
}

func renderTasks() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Tasks — 12 total (10 running, 2 failed)"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("%-12s %-20s %-10s %10s %10s %10s\n",
		styleTableHeader.Render("ID"),
		styleTableHeader.Render("SERVICE"),
		styleTableHeader.Render("NODE"),
		styleTableHeader.Render("STATUS"),
		styleTableHeader.Render("DESIRED"),
		styleTableHeader.Render("UPTIME"),
	))
	sb.WriteString(styleMuted.Render(strings.Repeat("─", 75)))
	sb.WriteString("\n")

	for _, t := range mockTasks {
		status := t.status
		if t.status == "running" {
			status = styleSuccess.Render("running")
		} else {
			status = styleError.Render("failed")
		}
		sb.WriteString(fmt.Sprintf("%-12s %-20s %-10s %10s %10s %10s\n",
			t.id, t.service, t.node, status, t.desired, t.uptime,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(styleCard.Render(
		styleCardTitle.Render("Task Health") + "\n" +
			fmt.Sprintf("  Running: %s    Failed: %s    Restart rate: %s",
				styleSuccess.Render("10"), styleError.Render("2"),
				styleMuted.Render("0.3/h")) + "\n" +
			styleError.Render("  ✗ task-6 (worker-3) — failed, auto-restart pending") + "\n" +
			styleError.Render("  ✗ task-7 (batch-processor) — failed, no restart policy"),
	))

	return styleContent.Render(sb.String())
}

func renderAlerts() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Alerts — 6 active (2 critical, 3 warning, 1 info)"))
	sb.WriteString("\n\n")

	for _, a := range mockAlerts {
		var levelStyled string
		var icon string
		switch a.level {
		case "critical":
			levelStyled = styleError.Render("CRIT")
			icon = styleError.Render("✗")
		case "warning":
			levelStyled = styleWarning.Render("WARN")
			icon = styleWarning.Render("⚠")
		case "info":
			levelStyled = styleMuted.Render("INFO")
			icon = styleMuted.Render("ℹ")
		}

		sb.WriteString(fmt.Sprintf("%s %s [%s] %-20s %s\n",
			icon, a.time, levelStyled, a.service, a.message,
		))
	}

	sb.WriteString("\n")
	sb.WriteString(styleCard.Render(
		styleCardTitle.Render("Alert Summary") + "\n" +
			fmt.Sprintf("  Critical: %s    Warning: %s    Info: %s",
				styleError.Render("2"), styleWarning.Render("3"),
				styleMuted.Render("1")) + "\n" +
			styleMuted.Render("  Alerts come via SSE topics: events, metrics") + "\n" +
			styleMuted.Render("  No ack endpoint — alerts are read-only"),
	))

	return styleContent.Render(sb.String())
}

func renderRecommendations() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("Recommendations — 6 pending (2 high, 2 medium, 2 low risk)"))
	sb.WriteString("\n\n")

	for _, r := range mockRecommendations {
		var riskStyled, tierStyled string
		switch r.risk {
		case "high":
			riskStyled = styleError.Render("high")
		case "medium":
			riskStyled = styleWarning.Render("medium")
		default:
			riskStyled = styleSuccess.Render("low")
		}
		switch r.tier {
		case "conservative":
			tierStyled = styleMuted.Render("conservative")
		case "balanced":
			tierStyled = styleHighlight.Render("balanced")
		case "aggressive":
			tierStyled = styleWarning.Render("aggressive")
		}

		sb.WriteString(styleCard.Render(
			styleCardTitle.Render(r.service) +
				"  " + tierStyled + "  risk:" + riskStyled + "\n" +
				fmt.Sprintf("  CPU: %s    MEM: %s\n", r.cpu, r.mem) +
				styleMuted.Render("  "+r.reason),
		))
	}

	sb.WriteString(styleMuted.Render("  Apply with: resma recommendations apply <service> --confirm"))
	sb.WriteString("\n")

	return styleContent.Render(sb.String())
}

// ─── Utils ────────────────────────────────────────────────────────────────────

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
