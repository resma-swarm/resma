package cli

import (
	"github.com/spf13/cobra"
)

var changeLogService string
var changeLogLimit int

var changeLogCmd = &cobra.Command{
	Use:   "change-log",
	Short: "Show configuration change history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	changeLogCmd.Flags().StringVar(&changeLogService, "service", "", "Filter by service name")
	changeLogCmd.Flags().IntVar(&changeLogLimit, "limit", 50, "Maximum number of entries to show")
	rootCmd.AddCommand(changeLogCmd)
}
