package cli

import (
	"github.com/spf13/cobra"
)

var recommendationsCmd = &cobra.Command{
	Use:   "recommendations",
	Short: "Manage resource limit recommendations",
}

var recommendationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recommendations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var recommendationsShowCmd = &cobra.Command{
	Use:   "show [service]",
	Short: "Show recommendations for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var recommendationsTriggersCmd = &cobra.Command{
	Use:   "triggers",
	Short: "Show recommendation triggers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var recommendationsStorageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Show storage recommendations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var recommendationsRecalculateCmd = &cobra.Command{
	Use:   "recalculate [service]",
	Short: "Recalculate recommendations for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var recommendationsSimulateCmd = &cobra.Command{
	Use:   "simulate [service]",
	Short: "Simulate applying recommendations for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var recommendationsApplyConfirm bool

var recommendationsApplyCmd = &cobra.Command{
	Use:   "apply [service]",
	Short: "Apply recommendations for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	recommendationsApplyCmd.Flags().BoolVarP(&recommendationsApplyConfirm, "confirm", "y", false, "Skip confirmation prompt")

	recommendationsCmd.AddCommand(recommendationsListCmd)
	recommendationsCmd.AddCommand(recommendationsShowCmd)
	recommendationsCmd.AddCommand(recommendationsTriggersCmd)
	recommendationsCmd.AddCommand(recommendationsStorageCmd)
	recommendationsCmd.AddCommand(recommendationsRecalculateCmd)
	recommendationsCmd.AddCommand(recommendationsSimulateCmd)
	recommendationsCmd.AddCommand(recommendationsApplyCmd)
	rootCmd.AddCommand(recommendationsCmd)
}
