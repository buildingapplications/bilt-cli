package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bilt-dev/bilt-cli/internal/api"
	"github.com/bilt-dev/bilt-cli/internal/config"
	"github.com/bilt-dev/bilt-cli/internal/runner"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	verbose bool
	logger  *zap.Logger
	cfg     *config.Config
	run     *runner.Runner
)

// SetVersion sets the version info displayed by --version.
func SetVersion(version, commit, date string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("bilt %s (commit %s, built %s)\n", version, commit, date))
}

// SetBaseURL overrides the default API base URL (called from main with ldflags value).
func SetBaseURL(url string) {
	api.SetDefaultBaseURL(url)
}

var rootCmd = &cobra.Command{
	Use:   "bilt",
	Short: "Build and install your Bilt app on your iPhone",
	Long:  "Bilt CLI — build your AI-generated React Native app and install it on your iPhone.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error

		// Initialize config
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Initialize logger
		logger, err = initLogger()
		if err != nil {
			return fmt.Errorf("initializing logger: %w", err)
		}

		// Initialize runner
		run = &runner.Runner{
			Logger:  logger,
			Verbose: verbose,
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logger != nil {
			_ = logger.Sync()
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
}

func initLogger() (*zap.Logger, error) {
	logDir := filepath.Join(config.BiltDir(), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	logFile := filepath.Join(logDir, "bilt.log")

	// File encoder config
	fileEncoderCfg := zap.NewProductionEncoderConfig()
	fileEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderCfg)

	// File sink
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	fileSink := zapcore.AddSync(file)

	// Always log to file at Info level
	fileCore := zapcore.NewCore(fileEncoder, fileSink, zap.InfoLevel)

	cores := []zapcore.Core{fileCore}

	// If verbose, also log to stderr at Debug level
	if verbose {
		consoleEncoderCfg := zap.NewDevelopmentEncoderConfig()
		consoleEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderCfg)
		stderrSink := zapcore.AddSync(os.Stderr)
		consoleCore := zapcore.NewCore(consoleEncoder, stderrSink, zap.DebugLevel)
		cores = append(cores, consoleCore)
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core), nil
}

// pruneOldLogs removes old build log files, keeping the newest maxKeep.
func pruneOldLogs(maxKeep int) {
	logDir := filepath.Join(config.BiltDir(), "logs")
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	var buildLogs []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 6 && e.Name()[:6] == "build-" {
			buildLogs = append(buildLogs, e)
		}
	}

	if len(buildLogs) <= maxKeep {
		return
	}

	// DirEntry list from ReadDir is already sorted by name.
	// Build logs are named build-<slug>-<timestamp>.log so oldest come first.
	toRemove := buildLogs[:len(buildLogs)-maxKeep]
	for _, e := range toRemove {
		_ = os.Remove(filepath.Join(logDir, e.Name()))
	}
}
