// Command demo is a runnable showcase of the tuikit frame kit: a Frame with
// three pages exercising the numbered navigation, TabStrip sub-tabs, Panel,
// the layout helpers, and the InputCapturer guard.
package main

import (
	"fmt"
	"image/color"
	"os"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

func main() {
	frame := tuikit.New(
		tuikit.WithBrand("tuikit", "reusable TUI frame kit"),
		tuikit.WithPages(newPanelsPage(), newAboutPage(), newSearchPage()),
		tuikit.WithStatus(func() (string, tuikit.Level) {
			return "press 1/2/3 to switch pages", tuikit.LevelInfo
		}),
	)
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// --- Panels page: TabStrip sub-tabs + a focus-switchable Panel ---

type panelsPage struct {
	theme tuikit.Theme
	focus int
}

func newPanelsPage() *panelsPage { return &panelsPage{theme: tuikit.DefaultTheme()} }

func (p *panelsPage) Title() string { return "Panels" }

func (p *panelsPage) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "tab", "shift+tab":
			p.focus = (p.focus + 1) % 2
		}
	}
	return nil
}

func (p *panelsPage) View(width, height int) string {
	titles := []string{"Alpha", "Beta"}
	accents := []color.Color{p.theme.Cyan, p.theme.Green}
	strip := p.theme.TabStrip(titles, accents, p.focus)
	body := fmt.Sprintf(
		"Focused panel: %s\n\nTab / Shift+Tab  switch panels within this page\n1 / 2 / 3        switch pages",
		titles[p.focus],
	)
	panel := tuikit.Panel{
		Theme:   p.theme,
		Accent:  accents[p.focus],
		Focused: true,
		Width:   width,
		Height:  max(3, height-2),
	}.Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, strip, panel)
}

// --- About page: layout helpers ---

type aboutPage struct{ theme tuikit.Theme }

func newAboutPage() *aboutPage { return &aboutPage{theme: tuikit.DefaultTheme()} }

func (p *aboutPage) Title() string              { return "About" }
func (p *aboutPage) Update(tea.Msg) tea.Cmd     { return nil }

func (p *aboutPage) View(width, height int) string {
	t := p.theme
	rows := []string{
		t.StatusTitle("tuikit", "demo", t.Cyan, t.Green, width),
		tuikit.Field("Components", "Frame · TabStrip · Panel"),
		tuikit.Field("Theme", "DefaultTheme (swap via WithTheme)"),
		tuikit.Field("Pages", "plain 3-method interface"),
		t.Rule(width),
		t.MutedStyle().Render("A reusable Bubble Tea frame kit — numbered page nav, chip tabs, panels."),
	}
	return t.PanelStyle(t.Blue, false).Width(width).Height(max(3, height-2)).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// --- Search page: demonstrates InputCapturer (number keys type, not navigate) ---

type searchPage struct {
	theme tuikit.Theme
	input textinput.Model
}

func newSearchPage() *searchPage {
	in := textinput.New()
	in.Placeholder = "press / to focus, then type; Esc to blur"
	return &searchPage{theme: tuikit.DefaultTheme(), input: in}
}

func (p *searchPage) Title() string { return "Search" }

// CapturingInput makes the Frame hand number keys to this page while the field
// is focused, instead of switching pages.
func (p *searchPage) CapturingInput() bool { return p.input.Focused() }

func (p *searchPage) Update(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if p.input.Focused() {
		switch k.String() {
		case "enter", "esc":
			p.input.Blur()
			return nil
		default:
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return cmd
		}
	}
	if k.String() == "/" {
		return p.input.Focus()
	}
	return nil
}

func (p *searchPage) View(width, height int) string {
	t := p.theme
	field := t.PanelStyle(t.Cyan, p.input.Focused()).Width(width).Render(p.input.View())
	hint := t.MutedStyle().Render("While focused, digits type into the field — page nav is suppressed.")
	return lipgloss.JoinVertical(lipgloss.Left, field, "", hint)
}
