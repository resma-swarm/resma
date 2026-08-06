package tabs

import tea "github.com/charmbracelet/bubbletea"

// Tab is the interface that each dashboard tab must implement.
type Tab interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Tab, tea.Cmd)
	View() string
	Title() string
}
