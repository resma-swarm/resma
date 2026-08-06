package components

import "github.com/charmbracelet/lipgloss"

// AgentList renders a list of agents with their status and current task.
type AgentList struct {
	agents []AgentItem
	width  int
	style  lipgloss.Style
}

// AgentItem represents a single agent in the list.
type AgentItem struct {
	ID     string
	Name   string
	Status string
	Task   string
}

// NewAgentList creates a new AgentList component.
func NewAgentList() *AgentList {
	return &AgentList{}
}

// View renders the agent list as a string.
func (a AgentList) View() string {
	return ""
}
