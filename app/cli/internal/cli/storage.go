package cli

import (
	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Show storage usage and volume metrics",
}

var storageSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show storage usage summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var storageTrendRange string

var storageTrendCmd = &cobra.Command{
	Use:   "trend",
	Short: "Show storage usage trend over time",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var storageVolumesCmd = &cobra.Command{
	Use:   "volumes",
	Short: "List all volumes and their usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var storageVolumeCmd = &cobra.Command{
	Use:   "volume [volume]",
	Short: "Show detailed information about a volume",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	storageTrendCmd.Flags().StringVar(&storageTrendRange, "range", "7d", "Time range (e.g. 1d, 7d, 30d)")

	storageCmd.AddCommand(storageSummaryCmd)
	storageCmd.AddCommand(storageTrendCmd)
	storageCmd.AddCommand(storageVolumesCmd)
	storageCmd.AddCommand(storageVolumeCmd)
	rootCmd.AddCommand(storageCmd)
}
