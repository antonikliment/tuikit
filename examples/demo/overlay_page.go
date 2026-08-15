package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

// --- Overlay page: a modal popup pinned anywhere in the frame ---
//
// The page renders its own body first and then hands it to the Overlay, which
// composites the box on top. Nothing below has to make room: the page's layout
// is identical whether the popup is open or not.

type anchor struct {
	label string
	align tuikit.Alignment
}

// Every corner, edge, and the middle — the nine places a popup can be pinned.
var anchors = []anchor{
	{"Top · Left", tuikit.Alignment{Horizontal: lipgloss.Left, Vertical: lipgloss.Top}},
	{"Top · Center", tuikit.Alignment{Horizontal: lipgloss.Center, Vertical: lipgloss.Top}},
	{"Top · Right", tuikit.Alignment{Horizontal: lipgloss.Right, Vertical: lipgloss.Top}},
	{"Middle · Left", tuikit.Alignment{Horizontal: lipgloss.Left, Vertical: lipgloss.Center}},
	{"Middle · Center", tuikit.Alignment{Horizontal: lipgloss.Center, Vertical: lipgloss.Center}},
	{"Middle · Right", tuikit.Alignment{Horizontal: lipgloss.Right, Vertical: lipgloss.Center}},
	{"Bottom · Left", tuikit.Alignment{Horizontal: lipgloss.Left, Vertical: lipgloss.Bottom}},
	{"Bottom · Center", tuikit.Alignment{Horizontal: lipgloss.Center, Vertical: lipgloss.Bottom}},
	{"Bottom · Right", tuikit.Alignment{Horizontal: lipgloss.Right, Vertical: lipgloss.Bottom}},
}

var (
	openKey  = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open"))
	pinKey   = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pin"))
	shiftKey = key.NewBinding(key.WithKeys("h", "j", "k", "l"), key.WithHelp("hjkl", "shift"))
)

type overlayPage struct {
	theme   tuikit.Theme
	overlay tuikit.Overlay
	at      int
}

func newOverlayPage() *overlayPage {
	t := tuikit.DefaultTheme()
	o := tuikit.NewOverlay(t)
	// Bindings the page acts on itself, advertised in the popup's hint row.
	o.Help = []key.Binding{pinKey, shiftKey}
	p := &overlayPage{theme: t, overlay: o, at: 4}
	p.open()
	return p
}

func (p *overlayPage) Title() string { return "Overlay" }

// CapturingInput keeps the Frame's number keys out of the way while the popup
// is up, the same guard the Search page uses for its input field.
func (p *overlayPage) CapturingInput() bool { return p.overlay.IsOpen() }

func (p *overlayPage) Update(msg tea.Msg) tea.Cmd {
	// The page claims its own keys before the Overlay, which is modal and would
	// otherwise swallow them.
	if k, ok := msg.(tea.KeyPressMsg); ok && p.pageKey(k) {
		return nil
	}
	p.overlay.Update(msg)
	return nil
}

func (p *overlayPage) pageKey(k tea.KeyPressMsg) bool {
	switch {
	case key.Matches(k, openKey):
		p.open()
	case key.Matches(k, pinKey):
		p.at = (p.at + 1) % len(anchors)
		p.open()
	case key.Matches(k, shiftKey):
		p.shift(k.String())
	default:
		return false
	}
	return true
}

// shift nudges the box off its anchor by a cell at a time; the Overlay clamps
// whatever this asks for to the frame.
func (p *overlayPage) shift(k string) {
	switch k {
	case "h":
		p.overlay.Align.ShiftX--
	case "l":
		p.overlay.Align.ShiftX++
	case "k":
		p.overlay.Align.ShiftY--
	case "j":
		p.overlay.Align.ShiftY++
	}
}

func (p *overlayPage) open() {
	at := anchors[p.at]
	shiftX, shiftY := p.overlay.Align.ShiftX, p.overlay.Align.ShiftY
	p.overlay.Align = at.align
	p.overlay.Align.ShiftX, p.overlay.Align.ShiftY = shiftX, shiftY
	p.overlay.Open("Pinned: "+at.label, overlayBody)
}

func (p *overlayPage) View(width, height int) string {
	t := p.theme
	rows := []string{
		t.StatusTitle("Overlay", "modal popup", t.Cyan, t.Green, width),
		tuikit.Field("Anchor", anchors[p.at].label),
		tuikit.Field("Shift", fmt.Sprintf("%+d, %+d", p.overlay.Align.ShiftX, p.overlay.Align.ShiftY)),
		t.Rule(width),
		t.MutedStyle().Render(strings.TrimSpace(`
The page draws this body once. The popup is composited on top of the finished
view, so nothing here reflows to make room for it — and when the popup closes,
the page is untouched.
`)),
		"",
		tuikit.HelpLine(openKey, pinKey, shiftKey),
	}
	body := t.PanelStyle(t.Blue, false).Width(width).Height(max(3, height-2)).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
	return p.overlay.Render(body, width, height)
}

var overlayBody = strings.TrimSpace(`
Overlay is a dismissable floating panel: a titled, bordered, scrollable box
drawn over the host's view rather than stacked above or below it.

It is where reference output belongs — a help table, a usage report, a list of
tools — that would otherwise be spliced into the main content stream, mixed in
with everything else, and scrolled away.

  o     reopen this popup
  p     pin it to the next of nine anchors
  hjkl  nudge it a cell off that anchor
  esc   close

The hint row below is always drawn: a modal with no visible way out is a trap.
Bindings the host puts in Help are appended to it — p and hjkl above are the
page's own keys, not the Overlay's.
`)
