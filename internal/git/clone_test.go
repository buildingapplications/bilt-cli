package git

import (
	"context"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/bilt-dev/bilt-cli/internal/runner"
)

func TestSplitCredential(t *testing.T) {
	cases := []struct {
		name, cloneURL, gitURL, plain, token string
	}{
		{"token as username", "https://tok@git.bilt.me/a/r.git", "https://git.bilt.me/a/r.git", "https://git.bilt.me/a/r.git", "tok"},
		{"token as password", "https://user:tok@git.bilt.me/a/r.git", "", "https://git.bilt.me/a/r.git", "tok"},
		{"no credential", "https://github.com/a/r.git", "", "https://github.com/a/r.git", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plain, token := splitCredential(c.cloneURL, c.gitURL)
			assert.Equal(t, c.plain, plain)
			assert.Equal(t, c.token, token)
		})
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return strings.TrimSpace(string(out))
}

// authServer serves repos over smart HTTP and accepts only `Authorization: token <want>`.
func authServer(t *testing.T, root string, want *atomic.Value) *httptest.Server {
	t.Helper()
	backend, err := exec.Command("git", "--exec-path").Output()
	require.NoError(t, err)
	handler := &cgi.Handler{
		Path: filepath.Join(strings.TrimSpace(string(backend)), "git-http-backend"),
		Env:  []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token "+want.Load().(string) {
			w.Header().Set("WWW-Authenticate", `Basic realm="gitea"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCloneAndPullUseOnlyThisBuildsToken(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	root := t.TempDir()
	seed := t.TempDir()
	git(t, root, "init", "-q", "--bare", "repo.git")
	git(t, seed, "init", "-q", "-b", "main")
	git(t, seed, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "one")
	git(t, seed, "push", "-q", filepath.Join(root, "repo.git"), "main")
	git(t, root, "--git-dir=repo.git", "symbolic-ref", "HEAD", "refs/heads/main")

	var want atomic.Value
	want.Store("first-token")
	server := authServer(t, root, &want)
	plain := server.URL + "/repo.git"
	withToken := func(tok string) string { return strings.Replace(plain, "http://", "http://"+tok+"@", 1) }
	r := &runner.Runner{Logger: zap.NewNop()}
	ctx := context.Background()

	dir, err := CloneOrUpdate(ctx, r, "p", withToken("first-token"), plain)
	require.NoError(t, err)
	config, err := os.ReadFile(filepath.Join(dir, ".git", "config"))
	require.NoError(t, err)
	assert.NotContains(t, string(config), "first-token")
	assert.Equal(t, plain, git(t, dir, "remote", "get-url", "origin"))

	// A later build: the first token is gone, a new one is issued, and an old clone may
	// still carry a stale token in its origin.
	git(t, seed, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "two")
	git(t, seed, "push", "-q", filepath.Join(root, "repo.git"), "main")
	git(t, dir, "remote", "set-url", "origin", withToken("stale-token"))
	want.Store("second-token")

	_, err = CloneOrUpdate(ctx, r, "p", withToken("second-token"), plain)
	require.NoError(t, err)
	assert.Equal(t, "two", git(t, dir, "log", "-1", "--format=%s"))
	assert.Equal(t, plain, git(t, dir, "remote", "get-url", "origin"))
	config, err = os.ReadFile(filepath.Join(dir, ".git", "config"))
	require.NoError(t, err)
	assert.NotContains(t, string(config), "token")
}
