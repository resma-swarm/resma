package cli

import (
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run diagnostic tests",
}

var testSmokeCmd = &cobra.Command{
	Use:   "smoke",
	Short: "Run smoke tests against the RESMA API",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	testCmd.AddCommand(testSmokeCmd)
	rootCmd.AddCommand(testCmd)
}
