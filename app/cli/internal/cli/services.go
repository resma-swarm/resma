package cli

import (
	"github.com/spf13/cobra"
)

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Manage and inspect Docker Swarm services",
}

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all monitored services",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesInspectCmd = &cobra.Command{
	Use:   "inspect [service]",
	Short: "Show detailed information about a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesMetricsRange string

var servicesMetricsCmd = &cobra.Command{
	Use:   "metrics [service]",
	Short: "Show resource metrics for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesContainersCmd = &cobra.Command{
	Use:   "containers [service]",
	Short: "List containers for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesSparklinesCmd = &cobra.Command{
	Use:   "sparklines [service]",
	Short: "Show CPU and memory sparklines for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show health status of all services",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesArchiveConfirm bool

var servicesArchiveCmd = &cobra.Command{
	Use:   "archive [service]",
	Short: "Archive a service (stop monitoring and retain history)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var servicesRestoreConfirm bool

var servicesRestoreCmd = &cobra.Command{
	Use:   "restore [service]",
	Short: "Restore an archived service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	servicesMetricsCmd.Flags().StringVar(&servicesMetricsRange, "range", "7d", "Time range (e.g. 1d, 7d, 30d)")

	servicesArchiveCmd.Flags().BoolVarP(&servicesArchiveConfirm, "confirm", "y", false, "Skip confirmation prompt")
	servicesRestoreCmd.Flags().BoolVarP(&servicesRestoreConfirm, "confirm", "y", false, "Skip confirmation prompt")

	servicesCmd.AddCommand(servicesListCmd)
	servicesCmd.AddCommand(servicesInspectCmd)
	servicesCmd.AddCommand(servicesMetricsCmd)
	servicesCmd.AddCommand(servicesContainersCmd)
	servicesCmd.AddCommand(servicesSparklinesCmd)
	servicesCmd.AddCommand(servicesHealthCmd)
	servicesCmd.AddCommand(servicesArchiveCmd)
	servicesCmd.AddCommand(servicesRestoreCmd)
	rootCmd.AddCommand(servicesCmd)
}
