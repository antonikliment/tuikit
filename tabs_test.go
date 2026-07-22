package tuikit

import (
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTabStripShowsAllTitles(t *testing.T) {
	theme := DefaultTheme()
	accents := []color.Color{theme.Cyan, theme.Green, theme.Yellow}
	out := ansi.Strip(theme.TabStrip([]string{"Alpha", "Beta", "Gamma"}, accents, 1))
	for _, want := range []string{"Alpha", "Beta", "Gamma"} {
		if !strings.Contains(out, want) {
			t.Fatalf("strip missing %q: %q", want, out)
		}
	}
}

func TestTabStripActiveDiffersFromInactive(t *testing.T) {
	theme := DefaultTheme()
	titles := []string{"One", "Two"}
	accents := []color.Color{theme.Cyan, theme.Green}

	active0 := theme.TabStrip(titles, accents, 0)
	active1 := theme.TabStrip(titles, accents, 1)
	if active0 == active1 {
		t.Fatal("active index should change the rendered chips")
	}
	// The active chip carries styling (escape sequences) even though the plain
	// text is identical.
	if ansi.Strip(active0) != ansi.Strip(active1) {
		t.Fatal("only styling should differ, not the text")
	}
	if !strings.Contains(active0, "\x1b[") {
		t.Fatal("active chip should be styled with escape sequences")
	}
}
