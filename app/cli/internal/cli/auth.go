package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/resma-swarm/resma/app/cli/internal/client"
	"github.com/spf13/cobra"
)

var (
	authServerURL string
	authUsername  string
	authPassword  string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and user profile",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and store credentials",
	Long: `Authenticate with the RESMA API server and persist credentials locally.

Credentials are stored in ~/.config/resma/credentials.json (or %APPDATA%\resma\ on Windows).
All subsequent commands use these credentials automatically.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Defaults
		if authServerURL == "" {
			authServerURL = "http://localhost:8080"
		}

		// Interactive prompts for missing fields
		if authUsername == "" {
			fmt.Print("Username: ")
			fmt.Scanln(&authUsername)
			if authUsername == "" {
				return fmt.Errorf("username is required")
			}
		}
		if authPassword == "" {
			fmt.Print("Password: ")
			fmt.Scanln(&authPassword)
			if authPassword == "" {
				return fmt.Errorf("password is required")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		creds, err := client.Login(ctx, authServerURL, authUsername, authPassword)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Printf("\n✓ Authenticated as %s (role: %s)\n", creds.Username, creds.Role)
		fmt.Printf("  Server: %s\n", creds.ServerURL)
		fmt.Printf("  Token expires: %s\n", creds.ExpiresAt.Format(time.RFC3339))
		fmt.Printf("  Credentials saved to ~/.config/resma/credentials.json\n")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.Logout(); err != nil {
			return fmt.Errorf("logout: %w", err)
		}
		fmt.Println("✓ Credentials cleared")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := client.LoadCredentials()
		if err != nil {
			return fmt.Errorf("load credentials: %w", err)
		}
		if creds == nil {
			fmt.Println("Not authenticated — run 'resma auth login' to authenticate")
			return nil
		}

		fmt.Printf("Server:       %s\n", creds.ServerURL)
		fmt.Printf("Username:     %s\n", creds.Username)
		fmt.Printf("Role:         %s\n", creds.Role)
		fmt.Printf("Token type:   %s\n", creds.TokenType)
		fmt.Printf("Expires at:   %s\n", creds.ExpiresAt.Format(time.RFC3339))
		if creds.IsExpired() {
			if creds.IsRefreshable() {
				fmt.Println("Status:       Token expired (will auto-refresh on next request)")
			} else {
				fmt.Println("Status:       Token expired — run 'resma auth login' again")
			}
		} else {
			remaining := time.Until(creds.ExpiresAt).Round(time.Second)
			fmt.Printf("Status:       Valid (%s remaining)\n", remaining)
		}
		return nil
	},
}

var authMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current user profile (from API)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		api, err := client.LoadOrRefresh(ctx)
		if err != nil {
			return err
		}
		_ = api
		// TODO: GET /api/auth/me quando implementado no API
		fmt.Printf("Authenticated as: %s\n", api.Credentials().Username)
		return nil
	},
}

var authChangePasswordCmd = &cobra.Command{
	Use:   "change-password",
	Short: "Change the current user's password",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not yet implemented")
	},
}

var authProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show or update the current user's profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not yet implemented")
	},
}

var authOnboardingCmd = &cobra.Command{
	Use:   "onboarding",
	Short: "Run the initial onboarding wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not yet implemented — use the web UI at http://localhost:5173")
	},
}

func init() {
	authLoginCmd.Flags().StringVar(&authServerURL, "server", "", "RESMA API server URL (default: http://localhost:8080)")
	authLoginCmd.Flags().StringVar(&authUsername, "username", "", "Username (prompted if not provided)")
	authLoginCmd.Flags().StringVar(&authPassword, "password", "", "Password (prompted if not provided)")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authMeCmd)
	authCmd.AddCommand(authChangePasswordCmd)
	authCmd.AddCommand(authProfileCmd)
	authCmd.AddCommand(authOnboardingCmd)
	rootCmd.AddCommand(authCmd)
}

// suppress unused import warning
var _ = os.Stdout
