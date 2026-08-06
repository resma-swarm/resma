package cli

import (
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune old data from the database",
}

var pruneDryRun bool

var prunePreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview what would be pruned without making changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var pruneServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "Prune old service metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var pruneNodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Prune old node metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var pruneTasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Prune old task records",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var pruneMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Prune old metric data points",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var pruneChangeLogCmd = &cobra.Command{
	Use:   "change-log",
	Short: "Prune old change log entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var pruneVolumeMetricsCmd = &cobra.Command{
	Use:   "volume-metrics",
	Short: "Prune old volume metric data points",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	prunePreviewCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")
	pruneServicesCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")
	pruneNodesCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")
	pruneTasksCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")
	pruneMetricsCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")
	pruneChangeLogCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")
	pruneVolumeMetricsCmd.Flags().BoolVar(&pruneDryRun, "dry-run", true, "Dry run (no changes written)")

	pruneCmd.AddCommand(prunePreviewCmd)
	pruneCmd.AddCommand(pruneServicesCmd)
	pruneCmd.AddCommand(pruneNodesCmd)
	pruneCmd.AddCommand(pruneTasksCmd)
	pruneCmd.AddCommand(pruneMetricsCmd)
	pruneCmd.AddCommand(pruneChangeLogCmd)
	pruneCmd.AddCommand(pruneVolumeMetricsCmd)
	rootCmd.AddCommand(pruneCmd)
}
