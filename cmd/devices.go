package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bilt-dev/bilt-cli/internal/device"
	"github.com/bilt-dev/bilt-cli/internal/platform"
	"github.com/bilt-dev/bilt-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List connected iOS devices",
	RunE:  runDevices,
}

func init() {
	rootCmd.AddCommand(devicesCmd)
}

func runDevices(cmd *cobra.Command, args []string) error {
	if !platform.IsMacOS() {
		fmt.Print(ui.FormatError("macOS required",
			"Device detection requires macOS with Xcode installed"))
		return nil
	}

	devices, err := device.Detect(cmd.Context(), run)
	if err != nil {
		return fmt.Errorf("detecting devices: %w", err)
	}

	if jsonOut {
		data, _ := json.MarshalIndent(devices, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println()
	if len(devices) == 0 {
		fmt.Printf("  %s No iOS devices connected\n\n", ui.CrossMark)
		fmt.Println(ui.Hint("Connect your iPhone via USB and trust this computer"))
		fmt.Println(ui.Hint("Make sure your device is unlocked when you connect it"))
		fmt.Println()
		return nil
	}

	// Table header
	nameW, modelW, iosW, connW := len("DEVICE"), len("MODEL"), len("iOS"), len("CONNECTION")
	for _, d := range devices {
		if len(d.Name) > nameW {
			nameW = len(d.Name)
		}
		if len(d.Model) > modelW {
			modelW = len(d.Model)
		}
	}

	widths := []int{nameW, modelW, iosW, connW}
	fmt.Println(ui.TableHeaderRow(widths, []string{"DEVICE", "MODEL", "iOS", "CONNECTION", "UDID"}))

	for _, d := range devices {
		ios := d.IOSVersion
		if ios == "" {
			ios = "—"
		}
		fmt.Printf("  %-*s  %-*s  %-*s  %-*s  %s\n",
			nameW, d.Name,
			modelW, d.Model,
			iosW, ios,
			connW, d.Connection,
			ui.Muted.Render(d.UDID),
		)
	}

	fmt.Printf("\n  %s %s\n\n",
		ui.CheckMark,
		ui.Muted.Render(fmt.Sprintf("%d device(s) found", len(devices))))
	return nil
}
