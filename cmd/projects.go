package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bilt-dev/bilt-cli/internal/api"
	"github.com/bilt-dev/bilt-cli/pkg/ui"
	"github.com/spf13/cobra"
)

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

	// Build table
	nameW, updatedW, statusW := len("PROJECT"), len("LAST UPDATED"), len("STATUS")
	for _, p := range projects {
		if len(p.Name) > nameW {
			nameW = len(p.Name)
		}
	}

	// Header
	header := fmt.Sprintf("  %-*s  %-*s  %-*s",
		nameW, "PROJECT",
		updatedW, "LAST UPDATED",
		statusW, "STATUS",
	)
	fmt.Println(ui.TableHeader.Render(header))

	// Rows
	for _, p := range projects {
		updated := p.UpdatedAt
		if t, err := time.Parse(time.RFC3339, p.UpdatedAt); err == nil {
			updated = t.Format("Jan 02, 2006")
		}
		status := ui.Muted.Render("no code")
		if p.GitURL != "" {
			status = ui.Success.Render("ready")
		}
		fmt.Printf("  %-*s  %-*s  %s\n",
			nameW, p.Name,
			updatedW, updated,
			status,
		)
	}

	fmt.Printf("\n  %s %s\n\n",
		ui.CheckMark,
		ui.Muted.Render(fmt.Sprintf("%d project(s)", len(projects))))
	return nil
}
