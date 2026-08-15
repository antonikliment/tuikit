package tuikit

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func testOverlay() Overlay { return NewOverlay(DefaultTheme()) }

// background is a plain filler view of the given size, the stand-in for a
// host's finished frame.
func background(width, height int) string {
	line := strings.Repeat("x", width)
	rows := make([]string, height)
	for i := range rows {
		rows[i] = line
	}
	return strings.Join(rows, "\n")
}

func TestOverlayOpenAndClose(t *testing.T) {
	o := testOverlay()
	if o.IsOpen() {
		t.Fatalf("a new overlay should be closed")
	}
	o.Open("context", "body")
	if !o.IsOpen() || o.Title() != "context" {
		t.Fatalf("open overlay: got open=%v title=%q", o.IsOpen(), o.Title())
	}
	o.Close()
	if o.IsOpen() || o.Title() != "" {
		t.Fatalf("closed overlay: got open=%v title=%q", o.IsOpen(), o.Title())
	}
}

func TestOverlayUpdateClaimsKeysOnlyWhileOpen(t *testing.T) {
	o := testOverlay()
	if o.Update(keyMsg("esc")) {
		t.Fatalf("a closed overlay must not claim keys")
	}
	o.Open("help", "body")
	if !o.Update(keyMsg("a")) {
		t.Fatalf("an open overlay is modal and must claim stray keys")
	}
	if !o.Update(keyMsg("esc")) {
		t.Fatalf("esc should be claimed")
	}
	if o.IsOpen() {
		t.Fatalf("esc should close the overlay")
	}
}

func TestOverlayRenderKeepsFrameSize(t *testing.T) {
	const w, h = 60, 20
	bg := background(w, h)

	o := testOverlay()
	if got := o.Render(bg, w, h); got != bg {
		t.Fatalf("a closed overlay must render the background unchanged")
	}

	o.Open("context", "context: 12k / 200k tokens")
	out := ansi.Strip(o.Render(bg, w, h))
	if got := lipgloss.Height(out); got != h {
		t.Fatalf("frame height = %d, want %d", got, h)
	}
	for i, line := range strings.Split(out, "\n") {
		if got := ansi.StringWidth(line); got != w {
			t.Fatalf("line %d width = %d, want %d", i, got, w)
		}
	}
}

func TestOverlayRenderShowsTitleContentAndHintOverBackground(t *testing.T) {
	o := testOverlay()
	o.Open("context", "context: 12k / 200k tokens")
	out := ansi.Strip(o.Render(background(60, 20), 60, 20))

	for _, want := range []string{"context: 12k / 200k tokens", "esc close", "┌", "x"} {
		if !strings.Contains(out, want) {
			t.Fatalf("overlay render missing %q:\n%s", want, out)
		}
	}
}

// Content larger than the frame — the case that would push the box past the
// terminal edge if it were not clamped.
func TestOverlayRenderClampsOversizedContent(t *testing.T) {
	for _, size := range []struct{ w, h int }{{8, 4}, {40, 10}} {
		o := testOverlay()
		o.Open("help", strings.Repeat("a very long line that cannot possibly fit\n", 40))
		out := ansi.Strip(o.Render(background(size.w, size.h), size.w, size.h))
		if got := lipgloss.Height(out); got != size.h {
			t.Fatalf("%dx%d: height = %d", size.w, size.h, got)
		}
		for i, line := range strings.Split(out, "\n") {
			if got := ansi.StringWidth(line); got != size.w {
				t.Fatalf("%dx%d: line %d width = %d", size.w, size.h, i, got)
			}
		}
	}
}

// However cramped the frame, the box keeps its title, its hint row, and at
// least one row of text — squeezing any of them out would leave a popup with no
// visible way to dismiss it.
func TestOverlayAlwaysKeepsTitleHintAndOneTextRow(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {40, 8}, {30, 6}, {30, 5}, {24, 4}} {
		o := testOverlay()
		o.Open("help", "FIRSTROW\nsecond\nthird\nfourth\nfifth\nsixth")
		out := ansi.Strip(o.Render(background(size.w, size.h), size.w, size.h))
		for _, want := range []string{"help", "FIRSTROW", "esc close"} {
			if !strings.Contains(out, want) {
				t.Fatalf("%dx%d: missing %q:\n%s", size.w, size.h, want, out)
			}
		}
	}
}

// The divider is decoration, so it is the first row dropped when the frame is
// too short for the full chrome.
func TestOverlayDividerSeparatesContentFromHint(t *testing.T) {
	o := testOverlay()
	o.Open("context", "body")
	if got := ansi.Strip(o.Render(background(40, 20), 40, 20)); !strings.Contains(got, "─────") {
		t.Fatalf("expected a divider above the hint row:\n%s", got)
	}
	if got := ansi.Strip(o.Render(background(30, 5), 30, 5)); !strings.Contains(got, "esc close") {
		t.Fatalf("a cramped frame must keep the hint, dropping the divider:\n%s", got)
	}
}

func TestOverlayAlignmentPinsAndShiftsTheBox(t *testing.T) {
	const w, h = 60, 20
	boxAt := func(align Alignment) (col, row int) {
		o := testOverlay()
		o.Align = align
		o.Open("t", "body")
		for y, line := range strings.Split(ansi.Strip(o.Render(background(w, h), w, h)), "\n") {
			if i := strings.Index(line, "┌"); i >= 0 {
				return i, y
			}
		}
		return -1, -1
	}

	centerX, centerY := boxAt(Alignment{Horizontal: lipgloss.Center, Vertical: lipgloss.Center})
	topLeftX, topLeftY := boxAt(Alignment{Horizontal: lipgloss.Left, Vertical: lipgloss.Top})
	if topLeftX != 0 || topLeftY != 0 {
		t.Fatalf("top-left = (%d,%d), want (0,0)", topLeftX, topLeftY)
	}
	if centerX <= topLeftX || centerY <= topLeftY {
		t.Fatalf("centered = (%d,%d), should sit past top-left", centerX, centerY)
	}

	shiftedX, shiftedY := boxAt(Alignment{Horizontal: lipgloss.Left, Vertical: lipgloss.Top, ShiftX: 3, ShiftY: 2})
	if shiftedX != 3 || shiftedY != 2 {
		t.Fatalf("shifted = (%d,%d), want (3,2)", shiftedX, shiftedY)
	}

	// A shift past the edge is clamped, never drawn off-frame.
	clampedX, clampedY := boxAt(Alignment{Horizontal: lipgloss.Right, Vertical: lipgloss.Bottom, ShiftX: 99, ShiftY: 99})
	if clampedX+lipgloss.Width("┌") > w || clampedY >= h {
		t.Fatalf("clamped = (%d,%d), want inside %dx%d", clampedX, clampedY, w, h)
	}
}

func TestOverlayHintShowsHostBindings(t *testing.T) {
	o := testOverlay()
	o.Help = []key.Binding{key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh"))}
	o.Open("context", "short")

	out := ansi.Strip(o.Render(background(60, 20), 60, 20))
	if !strings.Contains(out, "esc close") || !strings.Contains(out, "r refresh") {
		t.Fatalf("hint row should carry both built-in and host bindings:\n%s", out)
	}
}

func TestOverlayHintMentionsScrollOnlyWhenScrollable(t *testing.T) {
	o := testOverlay()
	o.Open("help", "one line")
	if got := ansi.Strip(o.Render(background(60, 20), 60, 20)); strings.Contains(got, "scroll") {
		t.Fatalf("content that fits should not advertise scrolling:\n%s", got)
	}

	o.Open("help", strings.Repeat("line\n", 100))
	if got := ansi.Strip(o.Render(background(60, 20), 60, 20)); !strings.Contains(got, "scroll") {
		t.Fatalf("overflowing content should advertise scrolling:\n%s", got)
	}
}
