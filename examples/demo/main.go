// Command demo is a runnable showcase of the tuikit frame kit: a Frame with
// several pages exercising the numbered navigation, TabStrip sub-tabs, Panel,
// a scrolling viewport, the layout helpers, and the InputCapturer guard.
package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

func main() {
	frame := tuikit.New(
		tuikit.WithBrand("tuikit", "reusable TUI frame kit"),
		tuikit.WithPages(newPanelsPage(), newReaderPage(), newAboutPage(), newSearchPage()),
		tuikit.WithStatus(func() (string, tuikit.Level) {
			return "press 1-4 to switch pages", tuikit.LevelInfo
		}),
	)
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// --- Panels page: TabStrip sub-tabs (more than two) + a focus-switchable Panel ---

type panelsPage struct {
	theme   tuikit.Theme
	titles  []string
	accents []color.Color
	focus   int
}

func newPanelsPage() *panelsPage {
	t := tuikit.DefaultTheme()
	return &panelsPage{
		theme:   t,
		titles:  []string{"Alpha", "Beta", "Gamma", "Delta"},
		accents: []color.Color{t.Cyan, t.Green, t.Yellow, t.Blue},
	}
}

func (p *panelsPage) Title() string { return "Panels" }

func (p *panelsPage) Update(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	n := len(p.titles)
	switch k.String() {
	case "tab":
		p.focus = (p.focus + 1) % n
	case "shift+tab":
		p.focus = (p.focus + n - 1) % n
	}
	return nil
}

func (p *panelsPage) View(width, height int) string {
	strip := p.theme.TabStrip(p.titles, p.accents, p.focus)
	body := fmt.Sprintf(
		"Focused sub-tab: %s\n\nTab / Shift+Tab  cycle sub-tabs (%d of them)\n1 – 4            switch pages",
		p.titles[p.focus], len(p.titles),
	)
	panel := tuikit.Panel{
		Theme:   p.theme,
		Accent:  p.accents[p.focus],
		Focused: true,
		Width:   width,
		Height:  max(3, height-2),
	}.Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, strip, panel)
}

// --- Reader page: a scrolling viewport of lorem ipsum ---

type readerPage struct {
	theme tuikit.Theme
	vp    viewport.Model
}

func newReaderPage() *readerPage {
	vp := viewport.New()
	vp.SetContent(loremIpsum)
	return &readerPage{theme: tuikit.DefaultTheme(), vp: vp}
}

func (p *readerPage) Title() string { return "Reader" }

func (p *readerPage) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg)
	return cmd
}

func (p *readerPage) View(width, height int) string {
	t := p.theme
	bodyH := max(3, height-2)
	p.vp.SetWidth(max(1, width-4))  // panel border + padding
	p.vp.SetHeight(max(1, bodyH-2)) // panel border
	title := fmt.Sprintf("Lorem ipsum — ↑/↓ PgUp/PgDn to scroll (%3.0f%%)", p.vp.ScrollPercent()*100)
	strip := t.Accent(t.Cyan).Render(title)
	panel := t.PanelStyle(t.Cyan, false).Width(width).Height(bodyH).Render(p.vp.View())
	return lipgloss.JoinVertical(lipgloss.Left, strip, panel)
}

// --- About page: layout helpers ---

type aboutPage struct{ theme tuikit.Theme }

func newAboutPage() *aboutPage { return &aboutPage{theme: tuikit.DefaultTheme()} }

func (p *aboutPage) Title() string          { return "About" }
func (p *aboutPage) Update(tea.Msg) tea.Cmd { return nil }

func (p *aboutPage) View(width, height int) string {
	t := p.theme
	rows := []string{
		t.StatusTitle("tuikit", "demo", t.Cyan, t.Green, width),
		tuikit.Field("Components", "Frame · TabStrip · Panel · viewport"),
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
	items []string
}

func newSearchPage() *searchPage {
	in := textinput.New()
	in.Placeholder = "press / to focus, then type to filter; Esc to blur"
	return &searchPage{theme: tuikit.DefaultTheme(), input: in, items: searchItems}
}

func (p *searchPage) filtered() []string {
	query := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if query == "" {
		return p.items
	}
	out := make([]string, 0, len(p.items))
	for _, item := range p.items {
		if strings.Contains(strings.ToLower(item), query) {
			out = append(out, item)
		}
	}
	return out
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

	results := p.filtered()
	lines := results
	if len(results) == 0 {
		lines = []string{t.MutedStyle().Render("no matches")}
	}
	bodyH := max(3, height-5) // input panel (3) + hint (1) + spacing
	body := tuikit.VerticalSlice(strings.Join(lines, "\n"), 0, max(1, bodyH-2))
	panel := t.PanelStyle(t.Green, false).Width(width).Height(bodyH).Render(body)

	hint := t.MutedStyle().Render(fmt.Sprintf(
		"%d / %d match  •  / focus  •  Esc blur  •  digits type while focused",
		len(results), len(p.items),
	))
	return lipgloss.JoinVertical(lipgloss.Left, field, panel, hint)
}

var searchItems = []string{
	"apple", "apricot", "avocado", "banana", "blackberry", "blueberry",
	"cantaloupe", "cherry", "clementine", "cranberry", "date", "dragonfruit",
	"elderberry", "fig", "gooseberry", "grape", "grapefruit", "guava",
	"honeydew", "kiwi", "kumquat", "lemon", "lime", "lychee", "mango",
	"nectarine", "orange", "papaya", "passionfruit", "peach", "pear",
	"persimmon", "pineapple", "plum", "pomegranate", "raspberry",
	"strawberry", "tangerine", "watermelon",
}

var loremIpsum = strings.TrimSpace(strings.Repeat(`Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse
cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non
proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium
doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore
veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim
ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit.

`, 8))
