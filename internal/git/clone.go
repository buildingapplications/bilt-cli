package git

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bilt-dev/bilt-cli/internal/config"
	"github.com/bilt-dev/bilt-cli/internal/runner"
)

const authHeaderEnv = "BILT_GIT_AUTH_HEADER"

func projectDir(slug string) string {
	return filepath.Join(config.BiltDir(), "projects", slug)
}

// splitCredential separates the short-lived token the server embeds in cloneURL from the
// plain repository URL, so the token never lands in .git/config, argv or debug logs.
func splitCredential(cloneURL, gitURL string) (plain, token string) {
	u, err := url.Parse(cloneURL)
	if err != nil || u.User == nil {
		return cloneURL, ""
	}
	token = u.User.Username()
	if password, ok := u.User.Password(); ok {
		token = password
	}
	u.User = nil
	if gitURL != "" {
		return gitURL, token
	}
	return u.String(), token
}

// authenticated runs git with the token as a header scoped to this command and repository.
func authenticated(
	ctx context.Context, r *runner.Runner, dir, plain, token string, args ...string,
) error {
	if token == "" {
		_, _, err := r.Run(ctx, dir, "git", args...)
		return err
	}
	prefix := []string{
		"-c", "credential.helper=",
		"--config-env=http." + plain + ".extraHeader=" + authHeaderEnv,
	}
	env := []string{authHeaderEnv + "=Authorization: token " + token, "GIT_TERMINAL_PROMPT=0"}
	_, _, err := r.RunWithEnv(ctx, dir, env, "git", append(prefix, args...)...)
	return err
}

// CloneOrUpdate clones the repo if it doesn't exist, or pulls latest changes.
// Returns the project directory path.
func CloneOrUpdate(ctx context.Context, r *runner.Runner, slug, cloneURL, gitURL string) (string, error) {
	dir := projectDir(slug)
	plain, token := splitCredential(cloneURL, gitURL)

	gitDir := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		// The stored origin may carry an expired token from an older build; each build
		// brings a fresh one, so pull through the plain URL with this build's header.
		if _, _, err := r.Run(ctx, dir, "git", "remote", "set-url", "origin", plain); err != nil {
			return dir, fmt.Errorf("updating project remote: %w", err)
		}
		if err := authenticated(ctx, r, dir, plain, token, "pull", "--ff-only"); err != nil {
			return dir, fmt.Errorf("updating project: %w", err)
		}
		return dir, nil
	}

	// Clone fresh
	if err := os.MkdirAll(filepath.Dir(dir), 0755); err != nil {
		return "", fmt.Errorf("creating projects directory: %w", err)
	}

	if err := authenticated(ctx, r, "", plain, token, "clone", plain, dir); err != nil {
		return "", fmt.Errorf("cloning project: %w", err)
	}
	return dir, nil
}
