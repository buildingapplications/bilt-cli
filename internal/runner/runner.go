package runner

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// Runner wraps exec.Command with logging and verbose support.
// All packages that need to run external commands receive a *Runner.
type Runner struct {
	Logger  *zap.Logger
	Verbose bool
}

// Run executes a command and returns stdout, stderr, and any error.
func (r *Runner) Run(ctx context.Context, dir string, name string, args ...string) (string, string, error) {
	return r.RunWithEnv(ctx, dir, nil, name, args...)
}

// RunWithEnv is Run with extra environment variables. Their values are never logged, so
// secrets belong here rather than in args.
func (r *Runner) RunWithEnv(
	ctx context.Context, dir string, env []string, name string, args ...string,
) (string, string, error) {
	r.Logger.Debug("running command",
		zap.String("cmd", name),
		zap.Strings("args", args),
		zap.String("dir", dir),
		zap.Int("extraEnv", len(env)),
	)

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	var stdout, stderr bytes.Buffer

	if r.Verbose {
		cmd.Stdout = io.MultiWriter(&stdout, os.Stderr)
		cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()

	outStr := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	if err != nil {
		r.Logger.Debug("command failed",
			zap.String("cmd", name),
			zap.Error(err),
			zap.String("stderr", errStr),
		)
		return outStr, errStr, fmt.Errorf("running %s: %w", name, err)
	}

	r.Logger.Debug("command succeeded", zap.String("cmd", name))
	return outStr, errStr, nil
}

// RunWithLog executes a command and streams all output to a log file.
// Used for long-running commands like xcodebuild where we want full logs on disk.
func (r *Runner) RunWithLog(ctx context.Context, dir string, logFile *os.File, name string, args ...string) error {
	r.Logger.Debug("running command with log",
		zap.String("cmd", name),
		zap.Strings("args", args),
		zap.String("dir", dir),
		zap.String("logFile", logFile.Name()),
	)

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	if r.Verbose {
		cmd.Stdout = io.MultiWriter(logFile, os.Stderr)
		cmd.Stderr = io.MultiWriter(logFile, os.Stderr)
	} else {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s: %w", name, err)
	}
	return nil
}
