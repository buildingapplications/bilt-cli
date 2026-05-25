package main

import (
	"os"

	"github.com/bilt-dev/bilt-cli/cmd"
)

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	baseURL = ""
)

func main() {
	cmd.SetVersion(version, commit, date)
	if baseURL != "" {
		cmd.SetBaseURL(baseURL)
	}
	if envBaseURL := os.Getenv("BILT_BASE_URL"); envBaseURL != "" {
		cmd.SetBaseURL(envBaseURL)
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
