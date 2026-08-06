package cli

import (
	"github.com/spf13/cobra"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Manage and inspect Docker Swarm nodes",
}

var nodesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all nodes in the Swarm",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var nodesClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Show cluster-level information and capacity",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var nodesInspectCmd = &cobra.Command{
	Use:   "inspect [node]",
	Short: "Show detailed information about a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var nodesMetricsRange string

var nodesMetricsCmd = &cobra.Command{
	Use:   "metrics [node]",
	Short: "Show resource metrics for a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var nodesServicesCmd = &cobra.Command{
	Use:   "services [node]",
	Short: "List services running on a node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	nodesMetricsCmd.Flags().StringVar(&nodesMetricsRange, "range", "7d", "Time range (e.g. 1d, 7d, 30d)")

	nodesCmd.AddCommand(nodesListCmd)
	nodesCmd.AddCommand(nodesClusterCmd)
	nodesCmd.AddCommand(nodesInspectCmd)
	nodesCmd.AddCommand(nodesMetricsCmd)
	nodesCmd.AddCommand(nodesServicesCmd)
	rootCmd.AddCommand(nodesCmd)
}
