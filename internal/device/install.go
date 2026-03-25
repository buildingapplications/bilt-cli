package device

import (
	"context"
	"fmt"

	"github.com/bilt-dev/bilt-cli/internal/runner"
)

// Install installs an .ipa onto the device with the given UDID.
func Install(ctx context.Context, r *runner.Runner, udid string, ipaPath string) error {
	// Try xcrun devicectl first (Xcode 15+)
	_, _, err := r.Run(ctx, "", "xcrun", "devicectl", "device", "install", "app",
		"--device", udid, ipaPath)
	if err == nil {
		return nil
	}

	// Fallback to ios-deploy
	_, _, err = r.Run(ctx, "", "ios-deploy", "--id", udid, "--bundle", ipaPath)
	if err != nil {
		return fmt.Errorf("installing app: %w (tried devicectl and ios-deploy)", err)
	}
	return nil
}
