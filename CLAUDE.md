# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Bilt CLI is a Go CLI tool that builds React Native / Expo projects as iOS apps and installs them onto connected iPhones. Users get a short build code from bilt.me, the CLI exchanges it for project info via the server API, then clones the repo, installs dependencies, runs xcodebuild, and deploys to the device.

## Commands

```sh
task build           # build binary for current platform
task check           # lint + vet + test (same as pre-commit hook)
task lint            # golangci-lint run ./...
task test            # go test -race ./...
task fmt             # gofmt + goimports
go test ./internal/xcode/...   # run tests for a single package
```

Requires: Go 1.26+, [Task](https://taskfile.dev), golangci-lint, goimports.

Pre-commit hook (husky) runs lint + vet + test. Run `npm install` once to set up husky.

## Architecture

**Entry point:** `main.go` sets version/commit/baseURL (injected via ldflags) and delegates to `cmd.Execute()`.

**`cmd/`** — Cobra command tree. Currently just the root command and `build`. The build command:
1. Exchanges a short build code for a `buildPayload` via `GET /api/cli/auth/exchange?code=<code>`
2. Orchestrates a multi-step build pipeline with progress UI (spinner + step lines)

**`internal/runner/`** — `Runner` wraps `exec.Command` with zap logging and verbose mode. All packages that shell out receive a `*Runner` rather than calling `exec` directly. Two methods: `Run` (capture stdout/stderr) and `RunWithLog` (stream to log file).

**`internal/config/`** — YAML config at `~/.bilt/config.yaml`. Stores API key and per-project cached settings (team ID, scheme, workspace). Thread-safe via mutex.

**`internal/` domain packages** (each stateless, receives `*Runner`):
- `prereq/` — checks Xcode, Node 18+, CocoaPods, Git, device, signing identity
- `git/` — clone or pull project source into `~/.bilt/projects/<slug>/`
- `deps/` — JS dependency install (npm/yarn), Expo prebuild detection, CocoaPods install
- `xcode/` — workspace/scheme discovery, scheme filtering (strips React Native library schemes), `xcodebuild archive`, team ID patching in pbxproj, signing team detection from Xcode preferences
- `device/` — iOS device detection and app install (devicectl with ios-deploy fallback)
- `platform/` — `IsMacOS()` guard

**`pkg/ui/`** — Terminal UI components using charmbracelet/lipgloss and bubbletea: styled text, step progress lines, error formatting, interactive selector.

## Key patterns

- `baseURL` defaults to `localhost:3000` and is overridden via `-X main.baseURL=...` ldflags for production builds
- Device operations try `xcrun devicectl` (Xcode 15+) first, fall back to `ios-deploy`
- Build logs go to `~/.bilt/logs/build-<slug>-<timestamp>.log`, pruned to newest 20

## Conventions

- Commit messages follow conventional commits (`feat:`, `fix:`, `feat!:`) — versions are auto-bumped from these
- Tests use `testify/assert`
- Release via `task release` (goreleaser)
