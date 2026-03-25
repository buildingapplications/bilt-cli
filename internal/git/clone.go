package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bilt-dev/bilt-cli/internal/config"
	"github.com/bilt-dev/bilt-cli/internal/runner"
)

// ProjectDir returns the local directory for a project.
func ProjectDir(slug string) string {
	return filepath.Join(config.BiltDir(), "projects", slug)
}

// CloneOrUpdate clones the repo if it doesn't exist, or pulls latest changes.
// Returns the project directory path.
func CloneOrUpdate(ctx context.Context, r *runner.Runner, slug, gitURL string) (string, error) {
	dir := ProjectDir(slug)

	gitDir := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		// Already cloned — pull
		_, _, err := r.Run(ctx, dir, "git", "pull", "--ff-only")
		if err != nil {
			return dir, fmt.Errorf("updating project: %w", err)
		}
		return dir, nil
	}

	// Clone fresh
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return "", fmt.Errorf("creating projects directory: %w", err)
	}

	_, _, err := r.Run(ctx, "", "git", "clone", gitURL, dir)
	if err != nil {
		return "", fmt.Errorf("cloning project: %w", err)
	}
	return dir, nil
}
