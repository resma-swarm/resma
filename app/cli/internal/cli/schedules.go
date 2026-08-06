package cli

import (
	"github.com/spf13/cobra"
)

var schedulesCmd = &cobra.Command{
	Use:   "schedules",
	Short: "Manage scheduled recommendation recalculation jobs",
}

var schedulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all schedules",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var schedulesPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "Show pending scheduled jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var schedulesHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show schedule execution history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var schedulesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new schedule",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var schedulesCancelConfirm bool

var schedulesCancelCmd = &cobra.Command{
	Use:   "cancel [schedule]",
	Short: "Cancel a schedule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	schedulesCancelCmd.Flags().BoolVarP(&schedulesCancelConfirm, "confirm", "y", false, "Skip confirmation prompt")

	schedulesCmd.AddCommand(schedulesListCmd)
	schedulesCmd.AddCommand(schedulesPendingCmd)
	schedulesCmd.AddCommand(schedulesHistoryCmd)
	schedulesCmd.AddCommand(schedulesCreateCmd)
	schedulesCmd.AddCommand(schedulesCancelCmd)
	rootCmd.AddCommand(schedulesCmd)
}
