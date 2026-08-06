package cli

import (
	"github.com/resma-swarm/resma/app/cli/internal/tui"
	"github.com/spf13/cobra"
)

var monitorService string

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Open the interactive TUI monitor for a service",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func init() {
	monitorCmd.Flags().StringVar(&monitorService, "service", "", "Service to monitor")
	rootCmd.AddCommand(monitorCmd)
}
