// Command demo is a runnable showcase of the tuikit frame kit: a Frame with
// several pages exercising the numbered navigation, TabStrip sub-tabs, Panel,
// a scrolling viewport, the layout helpers, and the InputCapturer guard.
package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

func main() {
	frame := tuikit.New(
		tuikit.WithBrand("tuikit", "reusable TUI frame kit"),
		tuikit.WithPages(newPanelsPage(), newReaderPage(), newAboutPage(), newSearchPage(), newWidgetsPage()),
		tuikit.WithStatus(func() (string, tuikit.Level) {
			return "press 1-5 to switch pages", tuikit.LevelInfo
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
	body := fmt.Sprintf(
		"Focused sub-tab: %s\n\nTab / Shift+Tab  cycle sub-tabs (%d of them)\n1 – 4            switch pages",
		p.titles[p.focus], len(p.titles),
	)
	return p.theme.TabbedPanel(p.titles, p.accents, p.focus, width, max(6, height-1), body)
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

// --- Search page: SearchView + ActionRow + HelpLine ---

type searchPage struct {
	theme    tuikit.Theme
	search   tuikit.SearchView
	actions  []string
	selected int
}

func newSearchPage() *searchPage {
	search := tuikit.NewSearchView()
	search.SetLines(searchItems)
	return &searchPage{
		theme:   tuikit.DefaultTheme(),
		search:  search,
		actions: []string{"Open", "Copy", "Delete"},
	}
}

func (p *searchPage) Title() string { return "Search" }

// CapturingInput makes the Frame hand number keys to this page while the field
// is focused, instead of switching pages.
func (p *searchPage) CapturingInput() bool { return p.search.Searching() }

func (p *searchPage) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && !p.search.Searching() {
		switch k.String() {
		case "left":
			p.selected = (p.selected + len(p.actions) - 1) % len(p.actions)
		case "right":
			p.selected = (p.selected + 1) % len(p.actions)
		}
	}
	p.search.Update(msg)
	return nil
}

func (p *searchPage) View(width, height int) string {
	t := p.theme
	bodyH := max(3, height-5) // action row + panel + help line
	panel := t.PanelStyle(t.Green, p.search.Searching()).Width(width).Height(bodyH).
		Render(p.search.View(max(1, width-4), max(1, bodyH-2)))
	search := tuikit.Field("Search", p.search.InputView())
	actions := t.ActionRow(t.Cyan, p.selected, p.actions, !p.search.Searching())
	help := tuikit.HelpLine(searchKey, moveKey, clearKey)
	return lipgloss.JoinVertical(lipgloss.Left, search, panel, actions, help)
}

var (
	searchKey = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "Search"))
	moveKey   = key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "Action"))
	clearKey  = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Clear"))
)

var searchItems = []string{
	"apple", "apricot", "avocado", "banana", "blackberry", "blueberry",
	"cantaloupe", "cherry", "clementine", "cranberry", "date", "dragonfruit",
	"elderberry", "fig", "gooseberry", "grape", "grapefruit", "guava",
	"honeydew", "kiwi", "kumquat", "lemon", "lime", "lychee", "mango",
	"nectarine", "orange", "papaya", "passionfruit", "peach", "pear",
	"persimmon", "pineapple", "plum", "pomegranate", "raspberry",
	"strawberry", "tangerine", "watermelon",
}

// --- Widgets page: Meter, Status (press-again-to-confirm), and text helpers ---

type widgetsPage struct {
	theme  tuikit.Theme
	cpu    tuikit.Meter
	ram    tuikit.Meter
	disk   tuikit.Meter
	status tuikit.Status // delete confirmation for the model below
}

func newWidgetsPage() *widgetsPage {
	t := tuikit.DefaultTheme()
	return &widgetsPage{
		theme: t,
		cpu:   tuikit.NewMeter(20, t.Green),
		ram:   tuikit.NewMeter(20, t.Yellow),
		disk:  tuikit.NewMeter(20, t.Red),
	}
}

func (p *widgetsPage) Title() string { return "Widgets" }

func (p *widgetsPage) Update(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "d", "y": // arm on first press, "delete" on the second
		p.status.Confirm("model.gguf", k.String() == "y", func() tea.Cmd {
			p.status.SetResult(nil, "deleted model.gguf")
			return nil
		})
	case "esc":
		p.status.Disarm()
	}
	return nil
}

func (p *widgetsPage) View(width, height int) string {
	t := p.theme
	const longPath = "/home/user/.cache/huggingface/models/meta-llama/Llama-3-8B/model.gguf"

	rows := []string{
		t.StatusTitle("Widgets", "meter · status · text", t.Cyan, t.Green, width),
		"CPU  " + p.cpu.View(37) + "  37%",
		"RAM  " + p.ram.View(68) + "  68%",
		"Disk " + p.disk.View(91) + "  91%",
		t.Rule(width),
		tuikit.Field("Size", tuikit.FormatBytes(4_812_390_400)),
		tuikit.Field("Path", tuikit.TruncMiddle(longPath, max(10, width-14))),
		t.Rule(width),
	}
	if pending := p.status.Pending(); pending != "" {
		rows = append(rows, t.Accent(t.Yellow).Render("Delete "+pending+"? press d/y to confirm, esc to cancel"))
	} else {
		rows = append(rows, t.MutedStyle().Render("press d to delete model.gguf (asks again to confirm)"))
		rows = p.status.AppendRows(t, rows)
	}
	return t.PanelStyle(t.Cyan, false).Width(width).Height(max(3, height-2)).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
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
