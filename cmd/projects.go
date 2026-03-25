package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bilt-dev/bilt-cli/internal/api"
	"github.com/bilt-dev/bilt-cli/pkg/ui"
	"github.com/spf13/cobra"
)

// firstLine returns the first line of s (handles multiline names from API).
func firstLine(s string) string {
	if i := strings.IndexAny(s, "\n\r"); i >= 0 {
		return s[:i]
	}
	return s
}

// displayLen returns the length of the first line of s.
func displayLen(s string) int {
	return len(firstLine(s))
}

// truncate shortens s to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your Bilt projects",
	RunE:  runProjectsList,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
}

func runProjectsList(cmd *cobra.Command, args []string) error {
	if cfg.Auth.APIKey == "" {
		fmt.Println(ui.FormatError("Not logged in",
			"Run `bilt auth login` first"))
		return fmt.Errorf("not logged in")
	}

	client := api.NewClient(cfg.Auth.APIKey)
	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("fetching projects: %w", err)
	}

	if jsonOut {
		data, _ := json.MarshalIndent(projects, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println()
	if len(projects) == 0 {
		fmt.Printf("  %s No projects yet.\n\n", ui.Muted.Render("○"))
		fmt.Println(ui.Hint("Create your first app at https://bilt.me"))
		fmt.Println()
		return nil
	}

	// Build table — truncate long names to keep the table readable
	const maxNameW = 30
	nameW, updatedW := len("PROJECT"), len("LAST UPDATED")
	for _, p := range projects {
		n := displayLen(p.Name)
		if n > nameW {
			nameW = n
		}
	}
	if nameW > maxNameW {
		nameW = maxNameW
	}

	widths := []int{nameW, updatedW}
	fmt.Println(ui.TableHeaderRow(widths, []string{"PROJECT", "LAST UPDATED", "STATUS"}))

	for _, p := range projects {
		updated := p.UpdatedAt
		if t, err := time.Parse(time.RFC3339, p.UpdatedAt); err == nil {
			updated = t.Format("Jan 02, 2006")
		}
		status := ui.Muted.Render("no code")
		if p.GitURL != "" {
			status = ui.Success.Render("ready")
		}
		name := truncate(firstLine(p.Name), nameW)
		fmt.Printf("  %-*s  %-*s  %s\n",
			nameW, name,
			updatedW, updated,
			status,
		)
	}

	fmt.Printf("\n  %s %s\n\n",
		ui.CheckMark,
		ui.Muted.Render(fmt.Sprintf("%d project(s)", len(projects))))
	return nil
}
