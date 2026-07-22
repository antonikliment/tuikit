// Command themed showcases live theme switching: press "t" to cycle palettes.
// The Frame chrome re-themes via tuikit.SetTheme, and the pages follow by
// reading a shared *tuikit.Theme the app swaps under them.
package main

import (
	"fmt"
	"image/color"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

type namedTheme struct {
	name  string
	theme tuikit.Theme
}

func main() {
	themes := []namedTheme{
		{"default", tuikit.DefaultTheme()},
		{"ocean", oceanTheme()},
		{"sunset", sunsetTheme()},
	}
	idx := 0
	shared := themes[idx].theme // pages read &shared; the closure swaps it

	frame := tuikit.New(
		tuikit.WithBrand("tuikit", "theme switcher — press t"),
		tuikit.WithTheme(shared),
		tuikit.WithPages(newPalettePage(&shared), newPanelsPage(&shared)),
		tuikit.WithGlobalKeys(func(msg tea.KeyPressMsg) (tea.Cmd, bool) {
			if msg.String() != "t" {
				return nil, false
			}
			idx = (idx + 1) % len(themes)
			shared = themes[idx].theme
			return tuikit.SetTheme(shared), true
		}),
		tuikit.WithStatus(func() (string, tuikit.Level) {
			return fmt.Sprintf("theme: %s  (press t to cycle, 1/2 to switch pages)", themes[idx].name), tuikit.LevelInfo
		}),
	)
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func oceanTheme() tuikit.Theme {
	return tuikit.Theme{
		Green:       lipgloss.Color("43"),
		Blue:        lipgloss.Color("39"),
		Yellow:      lipgloss.Color("80"),
		Red:         lipgloss.Color("203"),
		Cyan:        lipgloss.Color("51"),
		Muted:       lipgloss.Color("66"),
		Brand:       lipgloss.Color("45"),
		TabActiveFg: lipgloss.Color("233"),
		FocusBorder: lipgloss.Color("51"),
	}
}

func sunsetTheme() tuikit.Theme {
	return tuikit.Theme{
		Green:       lipgloss.Color("150"),
		Blue:        lipgloss.Color("175"),
		Yellow:      lipgloss.Color("214"),
		Red:         lipgloss.Color("196"),
		Cyan:        lipgloss.Color("209"),
		Muted:       lipgloss.Color("245"),
		Brand:       lipgloss.Color("209"),
		TabActiveFg: lipgloss.Color("235"),
		FocusBorder: lipgloss.Color("214"),
	}
}

// --- Palette page: shows the active theme's colors ---

type palettePage struct{ theme *tuikit.Theme }

func newPalettePage(theme *tuikit.Theme) *palettePage { return &palettePage{theme: theme} }

func (p *palettePage) Title() string          { return "Palette" }
func (p *palettePage) Update(tea.Msg) tea.Cmd { return nil }

func (p *palettePage) View(width, height int) string {
	t := *p.theme
	swatch := func(name string, c color.Color) string {
		block := lipgloss.NewStyle().Background(c).Render("      ")
		return block + "  " + t.Accent(c).Render(name)
	}
	rows := []string{
		t.StatusTitle("Palette", "press t to change", t.Brand, t.Green, width),
		t.Rule(width),
		swatch("Green ", t.Green),
		swatch("Blue  ", t.Blue),
		swatch("Yellow", t.Yellow),
		swatch("Red   ", t.Red),
		swatch("Cyan  ", t.Cyan),
		swatch("Muted ", t.Muted),
		swatch("Brand ", t.Brand),
	}
	return t.PanelStyle(t.Cyan, false).Width(width).Height(max(3, height-2)).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// --- Panels page: chip sub-tabs, re-themed live ---

type panelsPage struct {
	theme  *tuikit.Theme
	titles []string
	focus  int
}

func newPanelsPage(theme *tuikit.Theme) *panelsPage {
	return &panelsPage{theme: theme, titles: []string{"Alpha", "Beta", "Gamma"}}
}

func (p *panelsPage) Title() string { return "Panels" }

func (p *panelsPage) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		n := len(p.titles)
		switch k.String() {
		case "tab":
			p.focus = (p.focus + 1) % n
		case "shift+tab":
			p.focus = (p.focus + n - 1) % n
		}
	}
	return nil
}

func (p *panelsPage) View(width, height int) string {
	t := *p.theme
	accents := []color.Color{t.Cyan, t.Green, t.Yellow}
	strip := t.TabStrip(p.titles, accents, p.focus)
	body := fmt.Sprintf("Focused sub-tab: %s\n\nTab cycles sub-tabs; press t to re-theme everything.", p.titles[p.focus])
	panel := tuikit.Panel{Theme: t, Accent: accents[p.focus], Width: width, Height: max(3, height-3)}.Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, strip, panel)
}
