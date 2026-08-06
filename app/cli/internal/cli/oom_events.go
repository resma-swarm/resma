package cli

import (
	"github.com/spf13/cobra"
)

var oomEventsService string
var oomEventsRange string

var oomEventsCmd = &cobra.Command{
	Use:   "oom-events",
	Short: "Show OOM (out-of-memory) events",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	oomEventsCmd.Flags().StringVar(&oomEventsService, "service", "", "Filter by service name")
	oomEventsCmd.Flags().StringVar(&oomEventsRange, "range", "7d", "Time range (e.g. 1d, 7d, 30d)")
	rootCmd.AddCommand(oomEventsCmd)
}
