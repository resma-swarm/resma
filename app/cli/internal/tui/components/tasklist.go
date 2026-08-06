package components

import "github.com/charmbracelet/lipgloss"

// TaskList renders a list of tasks with status and progress.
type TaskList struct {
	tasks []TaskItem
	width int
	style lipgloss.Style
}

// TaskItem represents a single task in the list.
type TaskItem struct {
	ID       string
	Name     string
	Status   string
	Progress float64
}

// NewTaskList creates a new TaskList component.
func NewTaskList() *TaskList {
	return &TaskList{}
}

// View renders the task list as a string.
func (t TaskList) View() string {
	return ""
}
