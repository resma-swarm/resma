package cli

import (
	"github.com/spf13/cobra"
)

var rollbackWatchesCmd = &cobra.Command{
	Use:   "rollback-watches",
	Short: "Manage rollback watches for services",
}

var rollbackWatchesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rollback watches",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var rollbackWatchesInspectCmd = &cobra.Command{
	Use:   "inspect [watch]",
	Short: "Show detailed information about a rollback watch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var rollbackWatchesRollbackConfirm bool

var rollbackWatchesRollbackCmd = &cobra.Command{
	Use:   "rollback [watch]",
	Short: "Trigger a rollback for a watched service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var rollbackWatchesCancelConfirm bool

var rollbackWatchesCancelCmd = &cobra.Command{
	Use:   "cancel [watch]",
	Short: "Cancel a rollback watch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	rollbackWatchesRollbackCmd.Flags().BoolVarP(&rollbackWatchesRollbackConfirm, "confirm", "y", false, "Skip confirmation prompt")
	rollbackWatchesCancelCmd.Flags().BoolVarP(&rollbackWatchesCancelConfirm, "confirm", "y", false, "Skip confirmation prompt")

	rollbackWatchesCmd.AddCommand(rollbackWatchesListCmd)
	rollbackWatchesCmd.AddCommand(rollbackWatchesInspectCmd)
	rollbackWatchesCmd.AddCommand(rollbackWatchesRollbackCmd)
	rollbackWatchesCmd.AddCommand(rollbackWatchesCancelCmd)
	rootCmd.AddCommand(rollbackWatchesCmd)
}
