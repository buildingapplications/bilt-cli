package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Brand colors
var (
	ColorPrimary   = lipgloss.Color("#6C5CE7") // Bilt purple
	ColorSuccess   = lipgloss.Color("#00B894") // Green
	ColorWarning   = lipgloss.Color("#FDCB6E") // Yellow
	ColorError     = lipgloss.Color("#FF7675") // Red
	ColorMuted     = lipgloss.Color("#636E72") // Gray
	ColorHighlight = lipgloss.Color("#A29BFE") // Light purple
	ColorWhite     = lipgloss.Color("#FAFAFA")
)

// Text styles
var (
	Bold      = lipgloss.NewStyle().Bold(true)
	Muted     = lipgloss.NewStyle().Foreground(ColorMuted)
	Success   = lipgloss.NewStyle().Foreground(ColorSuccess)
	Warning   = lipgloss.NewStyle().Foreground(ColorWarning)
	ErrorText = lipgloss.NewStyle().Foreground(ColorError)
	Highlight = lipgloss.NewStyle().Foreground(ColorHighlight)
	Primary   = lipgloss.NewStyle().Foreground(ColorPrimary)
)

// Status indicators
var (
	CheckMark = Success.Render("✓")
	CrossMark = ErrorText.Render("✗")
	WarnMark  = Warning.Render("⚠")
	Dot       = Muted.Render("·")
	Arrow     = Primary.Render("→")
)

// Table styles
var (
	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorMuted)

	TableCell = lipgloss.NewStyle().
			PaddingRight(2)
)

// Box styles
var (
	InfoBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	ErrorBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(0, 1)

	SuccessBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(0, 1)

	WarningBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(0, 1)
)

// Logo prints the Bilt banner
func Logo() string {
	return Primary.Bold(true).Render("bilt")
}

// Header renders a section header with a divider line.
func Header(title string) string {
	t := Bold.Render(title)
	return fmt.Sprintf("\n  %s\n", t)
}

// StepProgress returns a formatted step indicator like "Step 3/7"
func StepProgress(current, total int) string {
	return Muted.Render(fmt.Sprintf("[%d/%d]", current, total))
}

// StepLine renders a step with status indicator.
//
//	status: "done", "active", "pending", "fail", "warn"
func StepLine(current, total int, label, status string) string {
	progress := Muted.Render(fmt.Sprintf("[%d/%d]", current, total))

	var icon string
	var text string
	switch status {
	case "done":
		icon = CheckMark
		text = label
	case "active":
		icon = Primary.Render("›")
		text = Bold.Render(label)
	case "fail":
		icon = CrossMark
		text = ErrorText.Render(label)
	case "warn":
		icon = WarnMark
		text = Warning.Render(label)
	default: // pending
		icon = Muted.Render("○")
		text = Muted.Render(label)
	}

	return fmt.Sprintf("%s %s %s", progress, icon, text)
}

// Hint renders an indented hint line (muted, with arrow prefix).
func Hint(text string) string {
	return fmt.Sprintf("      %s %s", Arrow, Muted.Render(text))
}

// Divider renders a subtle horizontal divider.
func Divider() string {
	return fmt.Sprintf("  %s", Muted.Render(strings.Repeat("─", 50)))
}

// FormatError renders a prominent error with optional hints.
func FormatError(title string, hints ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s %s\n", CrossMark, ErrorText.Bold(true).Render(title))
	for _, h := range hints {
		fmt.Fprintf(&b, "    %s %s\n", Arrow, Muted.Render(h))
	}
	return b.String()
}

// FormatKeyValue renders a labeled value pair with consistent alignment.
func FormatKeyValue(label, value string, labelWidth int) string {
	padded := fmt.Sprintf("%-*s", labelWidth, label)
	return fmt.Sprintf("  %s %s", Muted.Render(padded), value)
}
