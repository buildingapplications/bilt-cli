package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/bilt-dev/bilt-cli/internal/api"
	"github.com/bilt-dev/bilt-cli/internal/config"
	"github.com/bilt-dev/bilt-cli/internal/deps"
	"github.com/bilt-dev/bilt-cli/internal/device"
	gitpkg "github.com/bilt-dev/bilt-cli/internal/git"
	"github.com/bilt-dev/bilt-cli/internal/platform"
	"github.com/bilt-dev/bilt-cli/internal/prereq"
	"github.com/bilt-dev/bilt-cli/internal/xcode"
	"github.com/bilt-dev/bilt-cli/pkg/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	flagProject     string
	flagDevice      string
	flagSkipInstall bool
	flagToken       string
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build your app and install it on your iPhone",
	Long:  "Build your Bilt project as an iOS app and optionally install it on a connected device.",
	RunE:  runBuild,
}

func init() {
	buildCmd.Flags().StringVar(&flagProject, "project", "", "Project ID (required)")
	buildCmd.Flags().StringVar(&flagDevice, "device", "", "Device UDID to install on")
	buildCmd.Flags().BoolVar(&flagSkipInstall, "skip-install", false, "Build only, don't install on device")
	buildCmd.Flags().StringVar(&flagToken, "token", "", "One-time token from bilt.me (required)")
	_ = buildCmd.MarkFlagRequired("project")
	_ = buildCmd.MarkFlagRequired("token")
	rootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	if !platform.IsMacOS() {
		fmt.Println(ui.FormatError("macOS required",
			"bilt build requires macOS with Xcode installed"))
		return fmt.Errorf("bilt build requires macOS with Xcode installed")
	}

	// Exchange the one-time token for an API key
	fmt.Println()
	fmt.Printf("  %s Authenticating...\n", ui.Arrow)
	unauthClient := api.NewClient("")
	resp, err := unauthClient.ExchangeToken(flagToken)
	if err != nil {
		fmt.Println(ui.FormatError("Authentication failed", err.Error()))
		return fmt.Errorf("token exchange failed: %w", err)
	}
	if err := cfg.SetAPIKey(resp.APIKey); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}
	if resp.Email != "" {
		fmt.Printf("  %s Logged in as %s\n", ui.CheckMark, ui.Bold.Render(resp.Email))
	} else {
		fmt.Printf("  %s Authenticated\n", ui.CheckMark)
	}

	client := api.NewClient(cfg.Auth.APIKey)
	return runLocalBuild(cmd, client)
}

func runLocalBuild(cmd *cobra.Command, client *api.Client) error {
	ctx := cmd.Context()
	totalSteps := 9

	fmt.Println()

	// ── Step 1: Fetch project ──────────────────────────────────────────
	printStep(1, totalSteps, "Fetching project", "active")
	detail, err := client.GetProject(flagProject)
	if err != nil {
		printStep(1, totalSteps, "Project not found", "fail")
		return fmt.Errorf("fetching project: %w", err)
	}
	printStep(1, totalSteps, fmt.Sprintf("Project: %s", ui.Highlight.Render(detail.Name)), "done")
	fmt.Println()

	cloneURL := detail.CloneURL
	if cloneURL == "" {
		cloneURL = detail.GitURL
	}
	if cloneURL == "" {
		fmt.Println(ui.FormatError("No git URL found",
			"Make sure your project has generated code at https://bilt.me"))
		return fmt.Errorf("project %q has no git URL", detail.Name)
	}

	projectKey := sanitizeDirName(detail.Name)
	buildDir := filepath.Join(config.BiltDir(), "builds", projectKey)
	logDir := filepath.Join(config.BiltDir(), "logs")
	_ = os.MkdirAll(buildDir, 0755)
	_ = os.MkdirAll(logDir, 0755)

	// ── Step 2: Prerequisites ───────────────────────────────────────────
	printStep(2, totalSteps, "Checking prerequisites", "active")
	results := prereq.CheckAll(ctx, run)
	hasFailure := false
	for _, r := range results {
		if r.Critical && !r.OK {
			fmt.Printf("      %s %s\n", ui.CrossMark, r.Detail)
			if r.FixHint != "" {
				fmt.Println(ui.Hint(r.FixHint))
			}
			hasFailure = true
		}
	}
	if hasFailure {
		printStep(2, totalSteps, "Prerequisites check failed", "fail")
		return fmt.Errorf("missing prerequisites — fix the issues above and try again")
	}
	printStep(2, totalSteps, "All prerequisites met", "done")
	fmt.Println()

	// ── Step 3: Clone / update ──────────────────────────────────────────
	printStep(3, totalSteps, "Fetching project source", "active")
	projectDir, err := withSpinner("Cloning repository...", func() (string, error) {
		return gitpkg.CloneOrUpdate(ctx, run, projectKey, cloneURL)
	})
	if err != nil {
		printStep(3, totalSteps, "Clone failed", "fail")
		return err
	}
	printStep(3, totalSteps, "Source ready", "done")
	fmt.Println()

	// Detect if this is an Expo project that needs prebuild
	needsPrebuild := deps.IsExpoProject(projectDir) && deps.NeedsExpoPrebuild(projectDir)
	if needsPrebuild {
		totalSteps = 10
	}

	// Create build log file
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("build-%s-%s.log", projectKey, timestamp))
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	step := 4

	// ── Step 4: Install JS dependencies ─────────────────────────────────
	printStep(step, totalSteps, "Installing dependencies", "active")
	_, err = withSpinner("Installing JS dependencies...", func() (string, error) {
		return "", deps.InstallJS(ctx, run, projectDir, logFile)
	})
	if err != nil {
		printStep(step, totalSteps, "Dependency install failed", "fail")
		return buildError("npm/yarn install failed", logPath, err)
	}
	printStep(step, totalSteps, "Dependencies installed", "done")
	fmt.Println()
	step++

	// ── Step 5 (Expo only): Generate native project ─────────────────────
	if needsPrebuild {
		printStep(step, totalSteps, "Generating iOS project", "active")
		_, err = withSpinner("Running expo prebuild...", func() (string, error) {
			return "", deps.ExpoPrebuild(ctx, run, projectDir, logFile)
		})
		if err != nil {
			printStep(step, totalSteps, "Expo prebuild failed", "fail")
			return buildError("expo prebuild failed", logPath, err)
		}
		printStep(step, totalSteps, "iOS project generated", "done")
		fmt.Println()
		step++
	}

	// ── Step N: Install CocoaPods ───────────────────────────────────────
	printStep(step, totalSteps, "Installing CocoaPods", "active")
	_, err = withSpinner("Running pod install...", func() (string, error) {
		return "", deps.InstallPods(ctx, run, projectDir, logFile)
	})
	if err != nil {
		printStep(step, totalSteps, "Pod install failed", "fail")
		return buildError("pod install failed", logPath, err)
	}
	printStep(step, totalSteps, "CocoaPods installed", "done")
	fmt.Println()
	step++

	// ── Step N+1: Detect workspace + scheme ─────────────────────────────
	printStep(step, totalSteps, "Detecting workspace", "active")
	workspace, err := xcode.FindWorkspace(projectDir)
	if err != nil {
		printStep(step, totalSteps, "Workspace detection failed", "fail")
		return fmt.Errorf("finding workspace: %w", err)
	}

	schemes, err := xcode.ListSchemes(ctx, run, projectDir, workspace)
	if err != nil {
		return fmt.Errorf("listing schemes: %w", err)
	}
	if len(schemes) == 0 {
		return fmt.Errorf("no schemes found in workspace")
	}

	scheme := cfg.GetProject(projectKey).Scheme
	if scheme == "" || !containsString(schemes, scheme) {
		scheme = xcode.PickAppScheme(schemes, workspace)
	}
	if scheme == "" {
		return fmt.Errorf("could not determine app scheme from workspace")
	}

	printStep(step, totalSteps,
		fmt.Sprintf("Workspace: %s, Scheme: %s",
			ui.Highlight.Render(workspace), ui.Highlight.Render(scheme)),
		"done")
	fmt.Println()
	step++

	// ── Detect signing team ──────────────────────────────────────────────
	printStep(step, totalSteps, "Detecting signing team", "active")
	teamID := cfg.GetProject(projectKey).TeamID

	if teamID == "" {
		xcodeTeams, teamErr := xcode.FindXcodeTeams()
		if teamErr != nil || len(xcodeTeams) == 0 {
			printStep(step, totalSteps, "No signing team found", "fail")
			fmt.Println(ui.Hint("Open Xcode → Settings → Accounts → Add your Apple ID"))
			return fmt.Errorf("no signing team configured in Xcode")
		}
		if len(xcodeTeams) == 1 {
			teamID = xcodeTeams[0].TeamID
			printStep(step, totalSteps,
				fmt.Sprintf("Team: %s %s",
					xcodeTeams[0].TeamName,
					ui.Muted.Render("("+teamID+")")),
				"done")
		} else {
			// Interactive team selection
			items := make([]ui.SelectItem, len(xcodeTeams))
			for i, t := range xcodeTeams {
				items[i] = ui.SelectItem{
					Label: t.TeamName,
					Desc:  t.TeamID,
				}
			}
			choice, selErr := ui.Select("Select a signing team:", items)
			if selErr != nil {
				return fmt.Errorf("selection failed: %w", selErr)
			}
			if choice < 0 {
				return fmt.Errorf("cancelled")
			}
			teamID = xcodeTeams[choice].TeamID
			printStep(step, totalSteps,
				fmt.Sprintf("Team: %s %s",
					xcodeTeams[choice].TeamName,
					ui.Muted.Render("("+teamID+")")),
				"done")
		}
	} else {
		printStep(step, totalSteps,
			fmt.Sprintf("Team: %s %s", teamID, ui.Muted.Render("(cached)")),
			"done")
	}

	fmt.Println()
	step++

	// ── Build (archive) ─────────────────────────────────────────────────
	if err := xcode.PatchTeamID(projectDir, teamID); err != nil {
		logger.Warn("failed to patch team ID in pbxproj", zap.Error(err))
	}

	printStep(step, totalSteps, "Building iOS app", "active")
	archivePath := filepath.Join(buildDir, "App.xcarchive")
	derivedDataPath := filepath.Join(config.BiltDir(), "derived-data", projectKey)

	buildStart := time.Now()
	_, err = withSpinner("Compiling — this may take a few minutes...", func() (string, error) {
		return "", xcode.Archive(ctx, run, xcode.ArchiveOptions{
			ProjectDir:      projectDir,
			Workspace:       workspace,
			Scheme:          scheme,
			TeamID:          teamID,
			ArchivePath:     archivePath,
			DerivedDataPath: derivedDataPath,
			LogFile:         logFile,
		})
	})
	buildDuration := time.Since(buildStart)
	if err != nil {
		printStep(step, totalSteps, "Build failed", "fail")
		return xcodeBuildError(logPath, err)
	}
	printStep(step, totalSteps, fmt.Sprintf("Built in %s", formatDuration(buildDuration)), "done")

	appPath, err := xcode.FindAppInArchive(archivePath)
	if err != nil {
		return fmt.Errorf("finding app in archive: %w", err)
	}
	fmt.Println()
	step++

	// ── Install on device ───────────────────────────────────────────────
	if flagSkipInstall {
		printSummary(detail.Name, detail.BundleID, appPath, buildDuration)
		return nil
	}

	printStep(step, totalSteps, "Installing on device", "active")
	targetUDID := flagDevice

	if targetUDID == "" {
		devices, devErr := device.Detect(ctx, run)
		if devErr != nil || len(devices) == 0 {
			printStep(step, totalSteps, "No device connected — skipping install", "warn")
			fmt.Println()
			printSummary(detail.Name, detail.BundleID, appPath, buildDuration)
			return nil
		}
		if len(devices) == 1 {
			targetUDID = devices[0].UDID
			fmt.Printf("      Installing on %s...\n", ui.Bold.Render(devices[0].Name))
		} else {
			items := make([]ui.SelectItem, len(devices))
			for i, d := range devices {
				items[i] = ui.SelectItem{
					Label: d.Name,
					Desc:  fmt.Sprintf("%s · %s", d.Model, d.Connection),
				}
			}
			choice, selErr := ui.Select("Select a device:", items)
			if selErr != nil {
				return fmt.Errorf("selection failed: %w", selErr)
			}
			if choice < 0 {
				return fmt.Errorf("cancelled")
			}
			targetUDID = devices[choice].UDID
		}
	}

	_, err = withSpinner("Installing on device...", func() (string, error) {
		return "", device.Install(ctx, run, targetUDID, appPath)
	})
	if err != nil {
		printStep(step, totalSteps, "Install failed", "fail")
		fmt.Println(ui.Hint("The app was built successfully — you can install it manually."))
		fmt.Println(ui.Hint(appPath))
	} else {
		printStep(step, totalSteps, "Installed! Open the app on your device", "done")
	}
	fmt.Println()

	printSummary(detail.Name, detail.BundleID, appPath, buildDuration)

	// Cache project config
	_ = cfg.SetProject(projectKey, config.ProjectConfig{
		LastBuild: time.Now(),
		TeamID:    teamID,
		Scheme:    scheme,
		Workspace: workspace,
	})

	pruneOldLogs(20)

	return nil
}

func printStep(current, total int, label, status string) {
	fmt.Println(ui.StepLine(current, total, label, status))
}

func printSummary(appName, bundleID, appPath string, buildDuration time.Duration) {
	var lines []string
	lines = append(lines, fmt.Sprintf("  %s  Build Complete", ui.Bold.Render("")))
	lines = append(lines, "")
	lines = append(lines, ui.FormatKeyValue("App", appName, 10))
	if bundleID != "" {
		lines = append(lines, ui.FormatKeyValue("Bundle", bundleID, 10))
	}
	lines = append(lines, ui.FormatKeyValue("Duration", formatDuration(buildDuration), 10))
	lines = append(lines, ui.FormatKeyValue("Output", ui.Muted.Render(appPath), 10))

	fmt.Println(ui.SuccessBox.Render(strings.Join(lines, "\n")))
	fmt.Println()
	fmt.Printf("  %s Free provisioning apps expire in 7 days. Run %s again to renew.\n",
		ui.WarnMark, ui.Bold.Render("bilt build"))
}

func buildError(what, logPath string, err error) error {
	fmt.Print(ui.FormatError(what,
		fmt.Sprintf("Full log: %s", logPath)))
	return fmt.Errorf("%s: %w", what, err)
}

func xcodeBuildError(logPath string, err error) error {
	hints := []string{}

	data, readErr := os.ReadFile(logPath)
	if readErr == nil {
		logContent := strings.ToLower(string(data))

		switch {
		case strings.Contains(logContent, "no account for team"):
			hints = append(hints,
				"The selected team ID isn't signed into Xcode.",
				"Open Xcode → Settings → Accounts and sign in with the matching Apple ID.")
		case strings.Contains(logContent, "no signing certificate"):
			hints = append(hints,
				"Sign into your Apple ID in Xcode: Settings → Accounts")
		case strings.Contains(logContent, "no profiles for"):
			hints = append(hints,
				"Xcode couldn't provision the app.",
				"Open the project in Xcode once to create a provisioning profile, then try again.")
		case strings.Contains(logContent, "provisioning profile"):
			hints = append(hints,
				"Make sure Developer Mode is enabled on your device.")
		}
	}

	hints = append(hints, fmt.Sprintf("Full log: %s", logPath))
	fmt.Print(ui.FormatError("xcodebuild failed", hints...))
	return fmt.Errorf("build failed: %w", err)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func sanitizeDirName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if result == "" {
		return "project"
	}
	return result
}

type spinnerModel struct {
	spinner spinner.Model
	message string
	done    bool
}

type spinnerDoneMsg struct{}

func newSpinnerModel(msg string) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	return spinnerModel{spinner: s, message: msg}
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(spinnerDoneMsg); ok {
		m.done = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("      %s %s", m.spinner.View(), ui.Muted.Render(m.message))
}

func withSpinner[T any](message string, fn func() (T, error)) (T, error) {
	var result T
	var fnErr error
	done := make(chan struct{})

	go func() {
		result, fnErr = fn()
		close(done)
	}()

	m := newSpinnerModel(message)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))

	go func() {
		<-done
		p.Send(spinnerDoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		<-done
	}

	return result, fnErr
}
