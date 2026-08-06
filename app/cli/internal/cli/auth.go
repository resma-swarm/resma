package cli

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and user profile",
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and store credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var authMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current user profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var authChangePasswordCmd = &cobra.Command{
	Use:   "change-password",
	Short: "Change the current user's password",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var authProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show or update the current user's profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

var authOnboardingCmd = &cobra.Command{
	Use:   "onboarding",
	Short: "Run the initial onboarding wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil // TODO: implement
	},
}

func init() {
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authMeCmd)
	authCmd.AddCommand(authChangePasswordCmd)
	authCmd.AddCommand(authProfileCmd)
	authCmd.AddCommand(authOnboardingCmd)
	rootCmd.AddCommand(authCmd)
}
