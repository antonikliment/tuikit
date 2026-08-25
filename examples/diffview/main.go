// Command diffview shows [tuikit.DiffView]: the same edit rendered unified and
// side-by-side, syntax-highlighted, with the changed words inside a modified
// line pair picked out.
//
//	go run ./examples/diffview
//
// Press tab to switch layouts, +/- to change the context lines, and q to quit.
//
// The thing to watch is the modified pairs. Line for line the two versions are
// almost identical — a renamed parameter, a widened signature, a changed format
// string — and a plain +/- diff leaves finding the difference to the reader.
// Narrow the terminal below 100 columns and the side-by-side layout falls back
// to unified on its own rather than truncating both halves.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

const before = `package greeter

import "fmt"

// Greet builds the greeting line.
func Greet(name string) string {
	if name == "" {
		name = "world"
	}
	return fmt.Sprintf("hello, %s", name)
}

func Shout(name string) string {
	return strings.ToUpper(Greet(name))
}
`

const after = `package greeter

import "fmt"

// Greet builds the greeting line, repeated count times.
func Greet(subject string, count int) string {
	if subject == "" {
		subject = "world"
	}
	if count > 1 {
		return fmt.Sprintf("hello, %s (x%d)", subject, count)
	}
	return fmt.Sprintf("hello, %s!", subject)
}

func Shout(subject string) string {
	return strings.ToUpper(Greet(subject, 1))
}
`

type model struct {
	theme   tuikit.Theme
	diff    *tuikit.DiffView
	split   bool
	context int
	width   int
	height  int
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.split = !m.split
		case "+", "=":
			m.setContext(m.context + 1)
		case "-":
			m.setContext(m.context - 1)
		}
	}
	return m, nil
}

// setContext re-points the builder, which drops its render cache — the only
// state the example has to keep in step.
func (m *model) setContext(n int) {
	m.context = max(0, n)
	m.diff.ContextLines(m.context)
}

func (m *model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}
	// The panel border and its padding are not the diff's to draw in.
	inner := m.width - 6
	body := m.diff.MaxLines(max(4, m.height-8)).Render(inner)
	layout := "unified"
	if m.split {
		body, layout = m.diff.RenderSplit(inner), "side-by-side"
		if inner < tuikit.SplitMinWidth {
			layout = "side-by-side (too narrow — showing unified)"
		}
	}
	footer := m.theme.Accent(m.theme.Muted).Render(fmt.Sprintf(
		"%s · %d lines of context · tab to switch · +/- context · q to quit", layout, m.context))
	// Fixed height: the side-by-side layout puts a modified pair on one row
	// where unified needs two, so the frame would otherwise shrink on the
	// switch and leave the previous render's tail on the screen.
	panel := m.theme.PanelStyle(m.theme.Brand, true).
		Width(m.width - 2).
		Height(m.height - 8).
		Render(body)
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, panel, "", footer))
	// The alt screen: switching layouts changes how many lines the diff needs,
	// and an inline frame that shrinks leaves the previous one's tail behind.
	v.AltScreen = true
	return v
}

func main() {
	theme := tuikit.DefaultTheme()
	m := &model{
		theme:   theme,
		context: 3,
		diff: tuikit.NewDiffView(theme).
			Before("greeter.go", before).
			After("greeter.go", after).
			ContextLines(3),
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
