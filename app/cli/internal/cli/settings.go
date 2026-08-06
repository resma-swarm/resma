package cli

import (
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage system settings",
}

var settingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var settingsUpdateCmd = &cobra.Command{
	Use:   "update [key]",
	Short: "Update a setting value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	settingsCmd.AddCommand(settingsListCmd)
	settingsCmd.AddCommand(settingsUpdateCmd)
	rootCmd.AddCommand(settingsCmd)
}
