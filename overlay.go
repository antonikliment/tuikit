package tuikit

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Overlay is a dismissable floating panel: a titled, bordered, scrollable box
// drawn *over* a host's view rather than stacked above or below it. It is the
// place for reference output — a help table, a usage report, a tool list — that
// a host would otherwise have to splice into its main content and later scroll
// away.
//
// A host keeps one Overlay, calls Open when a view is requested, offers keys to
// Update before its own handling, and wraps its finished view in Render. While
// closed, Update claims nothing and Render returns the background untouched, so
// an Overlay costs a host two lines and changes no existing behavior.
type Overlay struct {
	// Theme supplies the border, title, and hint colors.
	Theme Theme
	// Accent is the border color. Zero means the theme's Brand.
	Accent color.Color
	// Help is extra bindings to advertise in the hint row, after the built-in
	// close and scroll keys. The row is always drawn, so these cost no height;
	// acting on them is the host's job (see Update).
	Help []key.Binding
	// Align pins the box in the frame. NewOverlay centers it.
	Align Alignment

	title   string
	vp      viewport.Model
	open    bool
	content string
}

// Overlay geometry: the box sizes itself to its content, floored at the
// minimums so a one-line popup still reads as a box and capped at the frame so
// it always fits.
const (
	minOverlayWidth = 24
	// Border (2) + title row + divider + hint row + one row of text: the
	// smallest box that still shows something.
	minOverlayHigh = 6
	// Border (2) + horizontal padding (2).
	overlayHChrome = 4
	// Border (2) + title row (1) + divider (1) + hint row (1).
	overlayVChrome = 5
	// Cells of the host's view left showing on each side, so the popup reads as
	// floating over it rather than replacing it.
	overlayMargin = 1
)

// Alignment pins an Overlay in its frame: an edge (or center) on each axis,
// plus a nudge in cells. Positive Shift moves right and down. The box is kept
// inside the frame whatever the shift asks for.
type Alignment struct {
	Horizontal lipgloss.Position // lipgloss.Left, Center, Right
	Vertical   lipgloss.Position // lipgloss.Top, Center, Bottom
	ShiftX     int
	ShiftY     int
}

// NewOverlay returns a closed Overlay drawing from the given theme, centered.
func NewOverlay(theme Theme) Overlay {
	return Overlay{
		Theme: theme,
		Align: Alignment{Horizontal: lipgloss.Center, Vertical: lipgloss.Center},
		vp:    viewport.New(),
	}
}

// Open shows content under the given title, scrolled to the top. Calling it on
// an already-open Overlay replaces what is showing.
func (o *Overlay) Open(title, content string) {
	o.title, o.content, o.open = title, content, true
	o.vp.GotoTop()
}

// Close hides the Overlay and drops its content.
func (o *Overlay) Close() {
	o.title, o.content, o.open = "", "", false
}

// IsOpen reports whether the Overlay is showing.
func (o *Overlay) IsOpen() bool { return o.open }

// Title is the title the Overlay was opened with.
func (o *Overlay) Title() string { return o.title }

// Content is the text the Overlay was opened with, unwrapped.
func (o *Overlay) Content() string { return o.content }

// Update offers a message to an open Overlay and reports whether it was
// claimed. Esc and "q" close it; the arrow/page/home/end keys scroll. A closed
// Overlay claims nothing, so a host can put this first in its key chain.
func (o *Overlay) Update(msg tea.Msg) bool {
	if !o.open {
		return false
	}
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}
	if key.Matches(press, overlayKeys.close) {
		o.Close()
		return true
	}
	if !key.Matches(press, overlayKeys.scroll) {
		// Swallow every other key: an open overlay is modal, so stray typing
		// must not reach the host's input.
		return true
	}
	o.vp, _ = o.vp.Update(press)
	return true
}

var overlayKeys = struct{ close, scroll key.Binding }{
	close: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc", "close")),
	scroll: key.NewBinding(
		key.WithKeys("up", "down", "k", "j", "pgup", "pgdown", "home", "end"),
		key.WithHelp("↑↓", "scroll"),
	),
}

// Render composites the Overlay over bg, centered in a width×height frame. The
// result is exactly that size, so a host can swap it in for its own view
// without redoing any height math. A closed Overlay renders as bg.
func (o *Overlay) Render(bg string, width, height int) string {
	if !o.open || width <= 0 || height <= 0 {
		return bg
	}
	box := o.box(width, height)
	x := place(width-lipgloss.Width(box), o.Align.Horizontal, o.Align.ShiftX)
	y := place(height-lipgloss.Height(box), o.Align.Vertical, o.Align.ShiftY)
	// A Layer's X/Y only take effect through a Compositor — drawn straight onto
	// a Canvas it lands at the origin.
	layers := lipgloss.NewCompositor(
		lipgloss.NewLayer(bg).Z(0),
		lipgloss.NewLayer(box).X(x).Y(y).Z(1),
	)
	return lipgloss.NewCanvas(width, height).Compose(layers).Render()
}

// box draws the panel itself: title, scrolling content, and the dismiss hint.
func (o *Overlay) box(width, height int) string {
	// Grow to the content, then clamp: a short report gets a small box and a
	// wide table gets as much of the frame as it needs — short of the edges, so
	// the view underneath still frames the popup as floating. The vertical
	// chrome reserves the title and the "esc close" row, which never scroll.
	// The hint row counts toward the width: a narrow report must not truncate
	// the line telling the reader how to get out of it.
	want := max(lipgloss.Width(o.content), lipgloss.Width(o.title), lipgloss.Width(o.fullHint()))
	boxW := fitOverlay(want+overlayHChrome, minOverlayWidth, width)
	innerW := max(1, boxW-overlayHChrome)
	wrapped := ansi.Wrap(o.content, innerW, "")
	boxH := fitOverlay(lipgloss.Height(wrapped)+overlayVChrome, minOverlayHigh, height)
	// On a frame too short for the full chrome the divider is the first thing to
	// go: it is decoration, while the title, a row of text, and the hint are not.
	divider := boxH >= minOverlayHigh
	innerH := max(1, boxH-overlayVChrome+boolToInt(!divider))

	o.vp.SetWidth(innerW)
	o.vp.SetHeight(innerH)
	o.vp.SetContent(wrapped)

	accent := o.Accent
	if accent == nil {
		accent = o.Theme.Brand
	}
	rows := []string{
		o.Theme.Accent(accent).Bold(true).Render(ansi.Truncate(o.title, innerW, "…")),
		o.vp.View(),
	}
	if divider {
		// A muted rule separates the content from the keys that act on it, so a
		// dense report does not run straight into its own footer.
		rows = append(rows, o.Theme.MutedStyle().Render(strings.Repeat("─", innerW)))
	}
	rows = append(rows, ansi.Truncate(o.hint(), innerW, "…"))
	panel := Panel{Theme: o.Theme, Accent: accent, Width: boxW, Height: boxH}
	return panel.Render(strings.Join(rows, "\n"))
}

// hint is the always-drawn bottom row: the close key, the scroll keys once
// there is anything to scroll, then whatever the host put in Help.
func (o *Overlay) hint() string {
	bindings := []key.Binding{overlayKeys.close}
	if !o.vp.AtTop() || !o.vp.AtBottom() {
		bindings = append(bindings, overlayKeys.scroll)
	}
	return HelpLine(append(bindings, o.Help...)...)
}

// fullHint is the widest the hint row can get — used for sizing, before the
// viewport knows whether it scrolls.
func (o *Overlay) fullHint() string {
	return HelpLine(append([]key.Binding{overlayKeys.close, overlayKeys.scroll}, o.Help...)...)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// place turns leftover space on one axis into an offset for the box, then
// clamps so a shift can never push it off the frame.
func place(slack int, pos lipgloss.Position, shift int) int {
	return min(max(0, int(float64(max(0, slack))*float64(pos))+shift), max(0, slack))
}

// fitOverlay sizes one axis: grow to want, keep at least minimum, stay inside
// the frame's margin — and, when the frame is too small for even the minimum,
// take the whole frame rather than overflow it.
func fitOverlay(want, minimum, frame int) int {
	size := max(want, minimum)
	return max(1, min(size, max(minimum, frame-overlayMargin*2), frame))
}
