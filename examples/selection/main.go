// Command selection shows [tuikit.Selection]: a mouse drag over a rendered
// frame, painted in place and copied out as plain text.
//
//	go run ./examples/selection
//
// Drag across the text with the left button held. The highlight follows the
// drag, and releasing copies the range to the clipboard — the footer echoes
// what went there. Press q to quit.
//
// The point is that the app draws the selection itself. A terminal program that
// enables mouse tracking takes drag events away from the terminal, so the
// terminal's own selection stops working; without something like this, there is
// no way to copy anything off the screen.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

const sample = `Selection holds two cells and nothing else.

The frame is handed to Paint and Text on every call, so a
selection cannot go stale against content that has been
re-rendered underneath it.

Columns are display columns, so wide runes land where they
look like they land:  GolandProjects/tuikit ·  main`

type model struct {
	theme     tuikit.Theme
	selection tuikit.Selection
	frame     string
	copied    string
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.selection.Clear()
	case tea.MouseClickMsg:
		m.selection.Begin(tuikit.Cell{Row: msg.Y, Col: msg.X})
	case tea.MouseMotionMsg:
		m.selection.Extend(tuikit.Cell{Row: msg.Y, Col: msg.X})
	case tea.MouseReleaseMsg:
		if text := m.selection.Text(m.frame); text != "" {
			// One line, bounded: the echo is proof the copy happened, not a
			// second copy of the text.
			m.copied = tuikit.TruncMiddle(strings.ReplaceAll(text, "\n", " ⏎ "), 56)
			m.selection.Clear()
			return m, tea.SetClipboard(text)
		}
		m.selection.Clear()
	}
	return m, nil
}

func (m *model) View() tea.View {
	muted := m.theme.Accent(m.theme.Muted)
	body := m.theme.PanelStyle(m.theme.Brand, true).Width(64).Render(sample)
	footer := muted.Render("drag to select · q to quit")
	if m.copied != "" {
		footer = m.theme.Accent(m.theme.Green).Render("copied") + " " + muted.Render(m.copied)
	}
	m.frame = lipgloss.JoinVertical(lipgloss.Left, "", body, "", footer)

	// A selection reads as one block of color with legible text on it, the way
	// a terminal's own does — background alone leaves the original foreground
	// fighting it.
	highlight := lipgloss.NewStyle().Background(m.theme.Brand).Foreground(m.theme.TabActiveFg)
	v := tea.NewView(m.selection.Paint(m.frame, highlight))
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func main() {
	m := &model{theme: tuikit.DefaultTheme()}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
