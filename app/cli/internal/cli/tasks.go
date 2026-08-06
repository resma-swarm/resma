package cli

import (
	"github.com/spf13/cobra"
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Inspect Swarm tasks",
}

var tasksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Swarm tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var tasksShowCmd = &cobra.Command{
	Use:   "show [task]",
	Short: "Show detailed information about a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var tasksHistoryCmd = &cobra.Command{
	Use:   "history [service]",
	Short: "Show task history for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	tasksCmd.AddCommand(tasksListCmd)
	tasksCmd.AddCommand(tasksShowCmd)
	tasksCmd.AddCommand(tasksHistoryCmd)
	rootCmd.AddCommand(tasksCmd)
}
