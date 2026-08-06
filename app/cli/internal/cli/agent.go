package cli

import (
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage the local RESMA agent",
}

var agentStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show local agent status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var agentHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check local agent health",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentHealthCmd)
	rootCmd.AddCommand(agentCmd)
}
