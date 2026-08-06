package cli

import (
	"github.com/spf13/cobra"
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage resource limit templates",
}

var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var templatesInspectCmd = &cobra.Command{
	Use:   "inspect [template]",
	Short: "Show detailed information about a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var templatesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new template",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var templatesUpdateCmd = &cobra.Command{
	Use:   "update [template]",
	Short: "Update an existing template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var templatesDeleteConfirm bool

var templatesDeleteCmd = &cobra.Command{
	Use:   "delete [template]",
	Short: "Delete a template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var templatesApplyConfirm bool

var templatesApplyCmd = &cobra.Command{
	Use:   "apply [template] [service]",
	Short: "Apply a template to a service",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	templatesDeleteCmd.Flags().BoolVarP(&templatesDeleteConfirm, "confirm", "y", false, "Skip confirmation prompt")
	templatesApplyCmd.Flags().BoolVarP(&templatesApplyConfirm, "confirm", "y", false, "Skip confirmation prompt")

	templatesCmd.AddCommand(templatesListCmd)
	templatesCmd.AddCommand(templatesInspectCmd)
	templatesCmd.AddCommand(templatesCreateCmd)
	templatesCmd.AddCommand(templatesUpdateCmd)
	templatesCmd.AddCommand(templatesDeleteCmd)
	templatesCmd.AddCommand(templatesApplyCmd)
	rootCmd.AddCommand(templatesCmd)
}
