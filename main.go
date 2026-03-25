package main

import (
	"os"

	"github.com/bilt-dev/bilt-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
