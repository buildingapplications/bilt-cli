package cmd

import (
	"fmt"
	"time"

	"github.com/bilt-dev/bilt-cli/internal/api"
	"github.com/bilt-dev/bilt-cli/internal/platform"
	"github.com/bilt-dev/bilt-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Bilt account via browser",
	RunE:  runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE:  runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	// Step 1: Start the device auth flow
	client := api.NewClient("")
	startResp, err := client.StartAuth()
	if err != nil {
		return fmt.Errorf("starting login: %w", err)
	}

	// Step 2: Open browser
	authURL := fmt.Sprintf("%s/cli-auth?code=%s", client.BaseURL, startResp.Code)

	fmt.Println()
	fmt.Printf("  %s Opening browser to log in...\n", ui.Arrow)
	fmt.Printf("  %s\n\n", ui.Muted.Render(authURL))

	if err := platform.OpenBrowser(authURL); err != nil {
		fmt.Printf("  %s Could not open browser automatically.\n", ui.WarnMark)
		fmt.Printf("      Copy and paste the URL above into your browser.\n\n")
	}

	// Step 3: Poll for completion
	fmt.Printf("  %s Waiting for browser login...\n", ui.Muted.Render("⏳"))

	deadline := time.Now().Add(time.Duration(startResp.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		pollResp, err := client.PollAuth(startResp.Code)
		if err != nil {
			// 404 means expired
			fmt.Print(ui.FormatError("Login expired",
				"Please run `bilt auth login` again"))
			return fmt.Errorf("login expired")
		}

		if pollResp.Status == "complete" {
			// Save the API key
			if err := cfg.SetAPIKey(pollResp.APIKey); err != nil {
				return fmt.Errorf("saving credentials: %w", err)
			}

			fmt.Println()
			if pollResp.Email != "" {
				fmt.Printf("  %s Logged in as %s\n\n", ui.CheckMark, ui.Bold.Render(pollResp.Email))
			} else {
				fmt.Printf("  %s Logged in successfully!\n\n", ui.CheckMark)
			}
			return nil
		}
		// Still pending — keep polling
	}

	fmt.Print(ui.FormatError("Login timed out",
		"Please run `bilt auth login` again"))
	return fmt.Errorf("login timed out")
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	if err := cfg.ClearAuth(); err != nil {
		return fmt.Errorf("clearing auth: %w", err)
	}
	fmt.Println()
	fmt.Printf("  %s Logged out successfully.\n\n", ui.CheckMark)
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	fmt.Println()
	if cfg.Auth.APIKey == "" {
		fmt.Printf("  %s Not logged in.\n", ui.CrossMark)
		fmt.Println(ui.Hint("Run `bilt auth login` to authenticate"))
		fmt.Println()
		return nil
	}

	client := api.NewClient(cfg.Auth.APIKey)
	user, err := client.Me()
	if err != nil {
		prefix := cfg.Auth.APIKey[:min(12, len(cfg.Auth.APIKey))]
		fmt.Printf("  %s Authentication invalid or expired\n", ui.CrossMark)
		fmt.Printf("  %s\n", ui.FormatKeyValue("Key", ui.Muted.Render(prefix+"..."), 8))
		fmt.Println(ui.Hint("Run `bilt auth login` to re-authenticate"))
		fmt.Println()
		return nil
	}

	fmt.Println(ui.FormatKeyValue("Name", ui.Bold.Render(user.Name), 8))
	fmt.Println(ui.FormatKeyValue("Email", user.Email, 8))
	fmt.Println(ui.FormatKeyValue("Plan", ui.Highlight.Render(user.Plan), 8))
	fmt.Println()
	return nil
}
