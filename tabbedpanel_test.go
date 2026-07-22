package tuikit

import (
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestNotchedTopMiddle(t *testing.T) {
	got := []rune(notchedTop(20, 5, 4)) // tab occupies columns 5..8
	if len(got) != 20 {
		t.Fatalf("len = %d, want 20", len(got))
	}
	if got[0] != '┌' || got[19] != '┐' {
		t.Fatalf("corners = %q..%q", got[0], got[19])
	}
	if got[5] != '┘' {
		t.Fatalf("left flare = %q, want ┘", got[5])
	}
	if got[8] != '└' {
		t.Fatalf("right flare = %q, want └", got[8])
	}
	for c := 6; c <= 7; c++ {
		if got[c] != ' ' {
			t.Fatalf("notch interior at %d = %q, want space", c, got[c])
		}
	}
}

func TestNotchedTopFlushLeft(t *testing.T) {
	got := []rune(notchedTop(20, 0, 4)) // tab occupies columns 0..3
	if got[0] != '│' {
		t.Fatalf("flush-left start = %q, want │ (border continues down)", got[0])
	}
	if got[3] != '└' {
		t.Fatalf("right flare = %q, want └", got[3])
	}
	if got[19] != '┐' {
		t.Fatalf("right corner = %q, want ┐", got[19])
	}
}

func TestTabbedPanelShowsTabsAndBody(t *testing.T) {
	theme := DefaultTheme()
	accents := []color.Color{theme.Cyan, theme.Green, theme.Yellow}
	out := ansi.Strip(theme.TabbedPanel([]string{"A", "B", "C"}, accents, 1, 44, 8, "BODY-TEXT"))

	// Tabs, body, and the notch flares (active tab is in the middle).
	for _, want := range []string{"A", "B", "C", "BODY-TEXT", "┘", "└"} {
		if !strings.Contains(out, want) {
			t.Fatalf("TabbedPanel missing %q:\n%s", want, out)
		}
	}
}
