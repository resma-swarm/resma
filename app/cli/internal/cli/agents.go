package cli

import (
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage and inspect RESMA agents",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var agentsInspectCmd = &cobra.Command{
	Use:   "inspect [agent]",
	Short: "Show detailed information about an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsInspectCmd)
	rootCmd.AddCommand(agentsCmd)
}
