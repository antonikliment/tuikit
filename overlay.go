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

	title   string
	vp      viewport.Model
	open    bool
	content string
}

// Overlay geometry: the box takes overlayRatio of each terminal axis, floored
// at the minimums so it stays legible (and non-negative) on a tiny terminal.
const (
	overlayRatioNum = 4
	overlayRatioDen = 5
	minOverlayWidth = 20
	minOverlayHigh  = 5
	// Border (2) + horizontal padding (2).
	overlayHChrome = 4
	// Border (2) + title row (1) + hint row (1).
	overlayVChrome = 4
)

// NewOverlay returns a closed Overlay drawing from the given theme.
func NewOverlay(theme Theme) Overlay {
	return Overlay{Theme: theme, vp: viewport.New()}
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
	close:  key.NewBinding(key.WithKeys("esc", "q")),
	scroll: key.NewBinding(key.WithKeys("up", "down", "k", "j", "pgup", "pgdown", "home", "end")),
}

// Render composites the Overlay over bg, centered in a width×height frame. The
// result is exactly that size, so a host can swap it in for its own view
// without redoing any height math. A closed Overlay renders as bg.
func (o *Overlay) Render(bg string, width, height int) string {
	if !o.open || width <= 0 || height <= 0 {
		return bg
	}
	box := o.box(width, height)
	x := max(0, (width-lipgloss.Width(box))/2)
	y := max(0, (height-lipgloss.Height(box))/2)
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
	boxW := clampOverlay(width*overlayRatioNum/overlayRatioDen, minOverlayWidth, width)
	boxH := clampOverlay(height*overlayRatioNum/overlayRatioDen, minOverlayHigh, height)
	innerW, innerH := max(1, boxW-overlayHChrome), max(1, boxH-overlayVChrome)

	o.vp.SetWidth(innerW)
	o.vp.SetHeight(innerH)
	o.vp.SetContent(ansi.Wrap(o.content, innerW, ""))

	accent := o.Accent
	if accent == nil {
		accent = o.Theme.Brand
	}
	rows := []string{
		o.Theme.Accent(accent).Bold(true).Render(ansi.Truncate(o.title, innerW, "…")),
		o.vp.View(),
		o.Theme.SubtleStyle().Render(o.hint()),
	}
	panel := Panel{Theme: o.Theme, Accent: accent, Width: boxW - 2, Height: boxH - 2}
	return panel.Render(strings.Join(rows, "\n"))
}

// hint is the bottom row; it drops the scroll half once everything fits.
func (o *Overlay) hint() string {
	if o.vp.AtTop() && o.vp.AtBottom() {
		return "esc close"
	}
	return "esc close · ↑↓ scroll"
}

func clampOverlay(v, lo, hi int) int {
	if hi < lo {
		return hi
	}
	return min(max(v, lo), hi)
}
