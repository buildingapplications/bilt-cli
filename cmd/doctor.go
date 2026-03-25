package cmd

import (
	"fmt"

	"github.com/bilt-dev/bilt-cli/internal/platform"
	"github.com/bilt-dev/bilt-cli/internal/prereq"
	"github.com/bilt-dev/bilt-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check prerequisites for building iOS apps",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		fmt.Println(ui.Header("Checking prerequisites"))

		results := prereq.CheckAll(cmd.Context(), run)

		passed := 0
		failed := 0
		for _, r := range results {
			if r.OK {
				fmt.Printf("  %s %s\n", ui.CheckMark, r.Detail)
				passed++
			} else {
				fmt.Printf("  %s %s\n", ui.CrossMark, r.Detail)
				if r.FixHint != "" {
					fmt.Println(ui.Hint(r.FixHint))
				}
				failed++
			}
		}

		fmt.Println()

		if !platform.IsMacOS() {
			fmt.Printf("  %s bilt build requires macOS with Xcode installed.\n\n", ui.WarnMark)
		}

		if prereq.HasCriticalFailures(results) {
			fmt.Printf("  %s %d passed, %s\n\n",
				ui.CrossMark,
				passed,
				ui.ErrorText.Render(fmt.Sprintf("%d failed", failed)))
			return fmt.Errorf("some critical prerequisites are missing")
		}

		fmt.Printf("  %s %s\n\n",
			ui.CheckMark,
			ui.Success.Render(fmt.Sprintf("All %d checks passed — ready to build!", passed)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
