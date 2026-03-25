package deps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bilt-dev/bilt-cli/internal/runner"
)

// InstallJS runs npm or yarn install in the project directory.
func InstallJS(ctx context.Context, r *runner.Runner, projectDir string, logFile *os.File) error {
	yarnLock := filepath.Join(projectDir, "yarn.lock")
	if _, err := os.Stat(yarnLock); err == nil {
		return r.RunWithLog(ctx, projectDir, logFile, "yarn", "install", "--frozen-lockfile")
	}

	packageLock := filepath.Join(projectDir, "package-lock.json")
	if _, err := os.Stat(packageLock); err == nil {
		return r.RunWithLog(ctx, projectDir, logFile, "npm", "ci")
	}

	return r.RunWithLog(ctx, projectDir, logFile, "npm", "install")
}

// IsExpoProject returns true if the project uses Expo (has app.json or app.config.ts/js).
func IsExpoProject(projectDir string) bool {
	for _, name := range []string{"app.json", "app.config.ts", "app.config.js"} {
		if _, err := os.Stat(filepath.Join(projectDir, name)); err == nil {
			return true
		}
	}
	return false
}

// NeedsExpoPrebuild returns true if the ios/ directory doesn't exist yet (Expo managed workflow).
func NeedsExpoPrebuild(projectDir string) bool {
	iosDir := filepath.Join(projectDir, "ios")
	_, err := os.Stat(iosDir)
	return os.IsNotExist(err)
}

// ExpoPrebuild runs `npx expo prebuild --platform ios` to generate the native ios/ directory.
func ExpoPrebuild(ctx context.Context, r *runner.Runner, projectDir string, logFile *os.File) error {
	return r.RunWithLog(ctx, projectDir, logFile, "npx", "expo", "prebuild", "--platform", "ios", "--no-install")
}

// InstallPods runs pod install in the ios/ directory.
func InstallPods(ctx context.Context, r *runner.Runner, projectDir string, logFile *os.File) error {
	iosDir := filepath.Join(projectDir, "ios")
	if _, err := os.Stat(iosDir); os.IsNotExist(err) {
		return fmt.Errorf("ios/ directory not found in project — run expo prebuild first")
	}
	return r.RunWithLog(ctx, iosDir, logFile, "pod", "install")
}
