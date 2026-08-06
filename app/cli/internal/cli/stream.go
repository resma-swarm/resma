package cli

import (
	"github.com/spf13/cobra"
)

var streamCmd = &cobra.Command{
	Use:   "stream [topic]",
	Short: "Stream real-time events for a given topic via SSE",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	rootCmd.AddCommand(streamCmd)
}
