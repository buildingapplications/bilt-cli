package prereq

import (
	"context"
	"fmt"

	"github.com/bilt-dev/bilt-cli/internal/runner"
)

func hasHomebrew(ctx context.Context, r *runner.Runner) bool {
	_, _, err := r.Run(ctx, "", "brew", "--version")
	return err == nil
}

func brewInstall(ctx context.Context, r *runner.Runner, formula string) error {
	_, _, err := r.Run(ctx, "", "brew", "install", formula)
	if err != nil {
		return fmt.Errorf("brew install %s: %w", formula, err)
	}
	return nil
}

func fixNode(ctx context.Context, r *runner.Runner) error {
	if !hasHomebrew(ctx, r) {
		return fmt.Errorf("homebrew not found — install Node.js manually from https://nodejs.org/")
	}
	return brewInstall(ctx, r, "node")
}

func fixCocoaPods(ctx context.Context, r *runner.Runner) error {
	if !hasHomebrew(ctx, r) {
		return fmt.Errorf("homebrew not found — install CocoaPods manually: sudo gem install cocoapods")
	}
	return brewInstall(ctx, r, "cocoapods")
}

func fixGit(ctx context.Context, r *runner.Runner) error {
	if !hasHomebrew(ctx, r) {
		return fmt.Errorf("homebrew not found — install Git manually from https://git-scm.com/")
	}
	return brewInstall(ctx, r, "git")
}

// xcodeAppStoreID is the Apple App Store ID for Xcode.
const xcodeAppStoreID = "497799835"

func fixXcode(ctx context.Context, r *runner.Runner) error {
	_, _, err := r.Run(ctx, "", "open", fmt.Sprintf("macappstore://itunes.apple.com/app/id%s", xcodeAppStoreID))
	if err != nil {
		return fmt.Errorf("could not open App Store — install Xcode manually from the App Store")
	}
	return fmt.Errorf("xcode is opening in the App Store — install it and re-run bilt build")
}
