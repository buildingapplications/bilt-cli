package prereq

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bilt-dev/bilt-cli/internal/platform"
	"github.com/bilt-dev/bilt-cli/internal/runner"
)

// CheckResult represents the outcome of a single prerequisite check.
type CheckResult struct {
	Name     string // e.g. "Xcode"
	OK       bool
	Detail   string // e.g. "Xcode 16.2 installed"
	FixHint  string // e.g. "Install Xcode from the App Store"
	Critical bool   // If true, build cannot proceed without this
}

// CheckAll runs all prerequisite checks. Requires macOS.
func CheckAll(ctx context.Context, r *runner.Runner) []CheckResult {
	if !platform.IsMacOS() {
		return []CheckResult{{
			Name:     "macOS",
			OK:       false,
			Detail:   "Not running on macOS",
			FixHint:  "bilt build requires macOS with Xcode installed",
			Critical: true,
		}}
	}
	return checkMacOS(ctx, r)
}

func checkMacOS(ctx context.Context, r *runner.Runner) []CheckResult {
	var results []CheckResult

	results = append(results, checkXcode(ctx, r))
	results = append(results, checkNode(ctx, r))
	results = append(results, checkCocoaPods(ctx, r))
	results = append(results, checkGit(ctx, r))
	results = append(results, checkDevice(ctx, r))
	results = append(results, checkSigning(ctx, r))

	return results
}

func checkXcode(ctx context.Context, r *runner.Runner) CheckResult {
	result := CheckResult{Name: "Xcode", Critical: true}

	// Check xcode-select
	_, _, err := r.Run(ctx, "", "xcode-select", "-p")
	if err != nil {
		result.Detail = "Xcode command line tools not found"
		result.FixHint = "Install Xcode from the App Store, then run: xcode-select --install"
		return result
	}

	// Get version
	stdout, _, err := r.Run(ctx, "", "xcodebuild", "-version")
	if err != nil {
		result.Detail = "xcodebuild not available"
		result.FixHint = "Install Xcode from the App Store"
		return result
	}

	// First line is like "Xcode 16.2"
	version := strings.SplitN(stdout, "\n", 2)[0]
	result.OK = true
	result.Detail = version + " installed"
	return result
}

func checkNode(ctx context.Context, r *runner.Runner) CheckResult {
	result := CheckResult{Name: "Node.js", Critical: true}

	stdout, _, err := r.Run(ctx, "", "node", "--version")
	if err != nil {
		result.Detail = "Node.js not found"
		result.FixHint = "Install Node.js 18+: https://nodejs.org/"
		return result
	}

	// stdout is like "v20.11.0"
	version := strings.TrimPrefix(strings.TrimSpace(stdout), "v")
	parts := strings.SplitN(version, ".", 2)
	if len(parts) >= 1 {
		major, parseErr := strconv.Atoi(parts[0])
		if parseErr == nil && major < 18 {
			result.Detail = fmt.Sprintf("Node.js v%s (need 18+)", version)
			result.FixHint = "Upgrade Node.js to version 18 or later: https://nodejs.org/"
			return result
		}
	}

	result.OK = true
	result.Detail = fmt.Sprintf("Node.js v%s installed", version)
	return result
}

func checkCocoaPods(ctx context.Context, r *runner.Runner) CheckResult {
	result := CheckResult{Name: "CocoaPods", Critical: true}

	stdout, _, err := r.Run(ctx, "", "pod", "--version")
	if err != nil {
		result.Detail = "CocoaPods not found"
		result.FixHint = "Install CocoaPods: sudo gem install cocoapods"
		return result
	}

	version := strings.TrimSpace(stdout)
	result.OK = true
	result.Detail = fmt.Sprintf("CocoaPods %s installed", version)
	return result
}

func checkGit(ctx context.Context, r *runner.Runner) CheckResult {
	result := CheckResult{Name: "Git", Critical: true}

	stdout, _, err := r.Run(ctx, "", "git", "--version")
	if err != nil {
		result.Detail = "Git not found"
		result.FixHint = "Install Git: https://git-scm.com/downloads"
		return result
	}

	// "git version 2.43.0"
	result.OK = true
	result.Detail = strings.TrimSpace(stdout) + " installed"
	return result
}

func checkDevice(ctx context.Context, r *runner.Runner) CheckResult {
	result := CheckResult{Name: "iOS Device", Critical: false}

	stdout, _, err := r.Run(ctx, "", "xcrun", "devicectl", "list", "devices", "--json-output", "/dev/stdout")
	if err != nil {
		// Fallback to ios-deploy
		stdout, _, err = r.Run(ctx, "", "ios-deploy", "--detect", "--timeout", "3")
		if err != nil {
			result.Detail = "No iOS device connected"
			result.FixHint = "Connect your iPhone via USB and trust this computer"
			return result
		}
		if strings.Contains(stdout, "Found") {
			result.OK = true
			result.Detail = "iOS device detected (via ios-deploy)"
			return result
		}
		result.Detail = "No iOS device connected"
		result.FixHint = "Connect your iPhone via USB and trust this computer"
		return result
	}

	// Parse devicectl JSON to count devices
	if strings.Contains(stdout, "deviceIdentifier") {
		result.OK = true
		result.Detail = "iOS device connected"
	} else {
		result.Detail = "No iOS device connected"
		result.FixHint = "Connect your iPhone via USB and trust this computer"
	}
	return result
}

func checkSigning(ctx context.Context, r *runner.Runner) CheckResult {
	result := CheckResult{Name: "Signing Identity", Critical: false}

	stdout, _, err := r.Run(ctx, "", "security", "find-identity", "-p", "codesigning", "-v")
	if err != nil {
		result.Detail = "Could not check signing identities"
		result.FixHint = "Open Xcode → Settings → Accounts → add your Apple ID"
		return result
	}

	if strings.Contains(stdout, "Apple Development") {
		// Count identities
		count := strings.Count(stdout, "Apple Development")
		result.OK = true
		if count == 1 {
			result.Detail = "1 Apple Development signing identity found"
		} else {
			result.Detail = fmt.Sprintf("%d Apple Development signing identities found", count)
		}
	} else {
		result.Detail = "No Apple Development signing identity"
		result.FixHint = "Open Xcode → Settings → Accounts → add your Apple ID"
	}
	return result
}

// HasCriticalFailures returns true if any critical check failed.
func HasCriticalFailures(results []CheckResult) bool {
	for _, r := range results {
		if r.Critical && !r.OK {
			return true
		}
	}
	return false
}
