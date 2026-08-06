package cli

import (
	"github.com/spf13/cobra"
)

var containersCmd = &cobra.Command{
	Use:   "containers",
	Short: "Inspect and monitor Docker containers",
}

var containersInspectCmd = &cobra.Command{
	Use:   "inspect [container]",
	Short: "Show detailed information about a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var containersMetricsRange string

var containersMetricsCmd = &cobra.Command{
	Use:   "metrics [container]",
	Short: "Show resource metrics for a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var containersNetworkCmd = &cobra.Command{
	Use:   "network [container]",
	Short: "Show network details for a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	containersMetricsCmd.Flags().StringVar(&containersMetricsRange, "range", "7d", "Time range (e.g. 1d, 7d, 30d)")

	containersCmd.AddCommand(containersInspectCmd)
	containersCmd.AddCommand(containersMetricsCmd)
	containersCmd.AddCommand(containersNetworkCmd)
	rootCmd.AddCommand(containersCmd)
}
