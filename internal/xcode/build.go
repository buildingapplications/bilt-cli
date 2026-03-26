package xcode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bilt-dev/bilt-cli/internal/runner"
)

// FindWorkspace finds the .xcworkspace file in the ios/ directory.
func FindWorkspace(projectDir string) (string, error) {
	iosDir := filepath.Join(projectDir, "ios")
	entries, err := os.ReadDir(iosDir)
	if err != nil {
		return "", fmt.Errorf("reading ios/ directory: %w", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xcworkspace") {
			return e.Name(), nil
		}
	}
	return "", fmt.Errorf("no .xcworkspace found in ios/")
}

// ListSchemes returns the available schemes for a workspace.
func ListSchemes(ctx context.Context, r *runner.Runner, projectDir, workspace string) ([]string, error) {
	wsPath := filepath.Join(projectDir, "ios", workspace)
	stdout, _, err := r.Run(ctx, "", "xcodebuild", "-workspace", wsPath, "-list")
	if err != nil {
		return nil, fmt.Errorf("listing schemes: %w", err)
	}

	return ParseSchemes(stdout), nil
}

// ParseSchemes extracts scheme names from xcodebuild -list output.
func ParseSchemes(output string) []string {
	var schemes []string
	inSchemes := false

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Schemes:") {
			inSchemes = true
			continue
		}

		// Stop at next section header (contains ":")
		if inSchemes && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "Schemes") {
			break
		}

		if inSchemes && trimmed != "" {
			schemes = append(schemes, trimmed)
		}
	}
	return schemes
}

// PickAppScheme selects the most likely app scheme from the list.
// It prefers: exact workspace name match > Pods-<Name> prefix match > shortest non-library name.
// Library schemes (React-*, RN*, Expo*, Pods-*, Yoga, hermes, etc.) are filtered out.
func PickAppScheme(schemes []string, workspaceName string) string {
	// Strip .xcworkspace extension for matching
	wsName := strings.TrimSuffix(workspaceName, ".xcworkspace")

	// First pass: exact match with workspace name
	for _, s := range schemes {
		if s == wsName {
			return s
		}
	}

	// Filter out known library/dependency schemes
	var candidates []string
	for _, s := range schemes {
		lower := strings.ToLower(s)
		if isLibraryScheme(lower) {
			continue
		}
		candidates = append(candidates, s)
	}

	if len(candidates) == 1 {
		return candidates[0]
	}
	if len(candidates) > 0 {
		return candidates[0]
	}

	// Fallback: return the workspace name if it exists at all, else first scheme
	if len(schemes) > 0 {
		return schemes[0]
	}
	return ""
}

func isLibraryScheme(lower string) bool {
	prefixes := []string{
		"react", "rn", "expo", "pods-", "hermes", "yoga", "fblazy",
		"rctrequired", "rctdeprecation", "rcttypesafety", "reactcodegen",
		"reactcommon", "reactnativedependencies", "reactappdependencyprovider",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	// Also filter schemes with hyphens that look like sub-targets (e.g. "EXConstants-EXConstants")
	// and known Expo internal modules
	expoInternal := []string{
		"exjsonutils", "exmanifests", "exupdatesinterface", "exconstants",
	}
	for _, e := range expoInternal {
		if strings.HasPrefix(lower, e) {
			return true
		}
	}
	return false
}

// Archive runs xcodebuild archive.
func Archive(ctx context.Context, r *runner.Runner, opts ArchiveOptions) error {
	wsPath := filepath.Join(opts.ProjectDir, "ios", opts.Workspace)

	args := []string{
		"archive",
		"-workspace", wsPath,
		"-scheme", opts.Scheme,
		"-configuration", "Release",
		"-destination", "generic/platform=iOS",
		"-archivePath", opts.ArchivePath,
		"-derivedDataPath", opts.DerivedDataPath,
		"-allowProvisioningUpdates",
		fmt.Sprintf("CODE_SIGN_IDENTITY=%s", "Apple Development"),
		fmt.Sprintf("DEVELOPMENT_TEAM=%s", opts.TeamID),
	}

	return r.RunWithLog(ctx, "", opts.LogFile, "xcodebuild", args...)
}

// ArchiveOptions configures the xcodebuild archive command.
type ArchiveOptions struct {
	ProjectDir      string
	Workspace       string
	Scheme          string
	TeamID          string
	ArchivePath     string
	DerivedDataPath string
	LogFile         *os.File
}

// FindAppInArchive finds the .app bundle inside a .xcarchive.
// Path is: <archive>/Products/Applications/<Name>.app
func FindAppInArchive(archivePath string) (string, error) {
	appsDir := filepath.Join(archivePath, "Products", "Applications")
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return "", fmt.Errorf("reading archive Applications: %w", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(appsDir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no .app found in archive")
}
