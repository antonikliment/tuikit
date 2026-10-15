// Command selection shows [tuikit.Selection]: a mouse drag over a rendered
// frame, painted in place and copied out as plain text.
//
//	go run ./examples/selection
//
// Drag across the text with the left button held. The highlight follows the
// drag, and releasing copies the range to the clipboard — the footer echoes
// what went there. Press q to quit.
//
// Hold alt while dragging for block mode: the selection is the rectangle the
// mouse drew, so a drag inside one of the three panels copies that panel alone
// instead of running through its neighbours to the ends of the lines.
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

// panels is the case linear selection gets wrong: a drag through the left one
// runs to the end of each line and picks up the other two.
var panels = []string{
	"logs\n12:01 boot ok\n12:02 listening\n12:03 request /x",
	"tools\nread_file    \u2713\nshell        \u2713\nfile_search  \u2715",
	"usage\nprompt 1.2k\noutput  340\ntotal   1.5k",
}

type model struct {
	theme     tuikit.Theme
	selection tuikit.Selection
	frame     string
	copied    string
	blockCopy bool
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
		// Alt-drag is the terminal convention for a rectangular selection.
		m.selection.Block = msg.Mod&tea.ModAlt != 0
		m.selection.Begin(tuikit.Cell{Row: msg.Y, Col: msg.X})
	case tea.MouseMotionMsg:
		m.selection.Extend(tuikit.Cell{Row: msg.Y, Col: msg.X})
	case tea.MouseReleaseMsg:
		if text := m.selection.Text(m.frame); text != "" {
			// One line, bounded: the echo is proof the copy happened, not a
			// second copy of the text.
			m.blockCopy = m.selection.Block
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
	row := make([]string, len(panels))
	for i, text := range panels {
		row[i] = m.theme.PanelStyle(m.theme.Muted, false).Width(20).Render(text)
	}
	columns := lipgloss.JoinHorizontal(lipgloss.Top, row...)
	footer := muted.Render("drag to select · alt-drag for a block · q to quit")
	if m.copied != "" {
		mode := "copied"
		if m.blockCopy {
			mode = "copied (block)"
		}
		footer = m.theme.Accent(m.theme.Green).Render(mode) + " " + muted.Render(m.copied)
	}
	m.frame = lipgloss.JoinVertical(lipgloss.Left, "", body, "", columns, "", footer)

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
