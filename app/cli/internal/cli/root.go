// Package cli contains all Cobra command definitions for resma-cli.
// Each file in this package corresponds to a command group (services, nodes, etc.).
// Commands are registered via init() functions and wired to the root command.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// rootCmd is the base command. All subcommands are attached to this.
var rootCmd = &cobra.Command{
	Use:   "resma",
	Short: "RESMA CLI — Resource Manager for Docker Swarm",
	Long: "resma-cli is the command-line interface for RESMA, a resource manager " +
		"for Docker Swarm. It provides inline commands for one-shot queries, " +
		"inline SSE streaming, and an interactive TUI dashboard (monitor).",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command. Called from main.go.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("resma: %w", err)
	}
	return nil
}
