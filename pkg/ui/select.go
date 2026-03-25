package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectItem represents a single option in the selector.
type SelectItem struct {
	Label string
	Desc  string // optional secondary line
}

type selectModel struct {
	items    []SelectItem
	cursor   int
	selected int
	title    string
	quitting bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "enter":
		m.selected = m.cursor
		m.quitting = true
		return m, tea.Quit
	case "ctrl+c", "q", "esc":
		m.selected = -1
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

var (
	selectCursor    = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	selectActive    = lipgloss.NewStyle().Foreground(ColorPrimary)
	selectInactive  = lipgloss.NewStyle().Foreground(ColorMuted)
	selectDescStyle = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)
)

func (m selectModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	if m.title != "" {
		fmt.Fprintf(&b, "  %s\n\n", Bold.Render(m.title))
	}

	for i, item := range m.items {
		if i == m.cursor {
			fmt.Fprintf(&b, "  %s %s\n",
				selectCursor.Render("›"),
				selectActive.Render(item.Label),
			)
		} else {
			fmt.Fprintf(&b, "    %s\n",
				selectInactive.Render(item.Label),
			)
		}
		if item.Desc != "" && i == m.cursor {
			fmt.Fprintf(&b, "    %s\n", selectDescStyle.Render(item.Desc))
		}
	}

	fmt.Fprintf(&b, "\n  %s",
		Muted.Render("↑/↓ navigate · enter select · esc cancel"))

	return b.String()
}

// Select presents an interactive selector and returns the chosen index.
// Returns -1 if the user cancels.
func Select(title string, items []SelectItem) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items to select from")
	}

	m := selectModel{
		items:    items,
		selected: -1,
		title:    title,
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return -1, err
	}

	final := result.(selectModel)
	return final.selected, nil
}

// SelectStrings is a convenience wrapper for simple string lists.
func SelectStrings(title string, labels []string) (int, error) {
	items := make([]SelectItem, len(labels))
	for i, l := range labels {
		items[i] = SelectItem{Label: l}
	}
	return Select(title, items)
}
