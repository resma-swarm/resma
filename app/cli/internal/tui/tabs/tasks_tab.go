package tabs

import tea "github.com/charmbracelet/bubbletea"

// TasksTab represents Tab [4] Tasks in the dashboard.
type TasksTab struct{}

// NewTasksTab creates a new TasksTab instance.
func NewTasksTab() *TasksTab {
	return &TasksTab{}
}

// Title returns the display title for this tab.
func (t TasksTab) Title() string {
	return "Tasks"
}

// Init performs initial setup for the tasks tab.
func (t TasksTab) Init() tea.Cmd {
	return nil
}

// Update handles messages for the tasks tab.
func (t TasksTab) Update(msg tea.Msg) (Tab, tea.Cmd) {
	return t, nil
}

// View renders the tasks tab as a string.
func (t TasksTab) View() string {
	return ""
}
