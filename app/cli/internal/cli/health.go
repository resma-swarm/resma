package cli

import (
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check RESMA API health",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Check RESMA API readiness",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(readyCmd)
}
