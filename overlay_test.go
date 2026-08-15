package tuikit

import (
	"strings"
	"testing"

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

func TestOverlayRenderSurvivesTinyFrame(t *testing.T) {
	o := testOverlay()
	o.Open("help", "a very long line that cannot possibly fit in here")
	out := ansi.Strip(o.Render(background(8, 4), 8, 4))
	if got := lipgloss.Height(out); got != 4 {
		t.Fatalf("tiny frame height = %d, want 4", got)
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
