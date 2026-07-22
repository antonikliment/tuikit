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

func TestTabStripActiveHasThreeSidedBorder(t *testing.T) {
	theme := DefaultTheme()
	accents := []color.Color{theme.Cyan, theme.Green}
	out := ansi.Strip(theme.TabStrip([]string{"One", "Two"}, accents, 0))

	// Active tab has a top border with corners (left/top/right) ...
	for _, want := range []string{"┌", "┐", "│"} {
		if !strings.Contains(out, want) {
			t.Fatalf("active tab missing border char %q:\n%s", want, out)
		}
	}
	// ... but is open at the bottom (no bottom corners).
	for _, absent := range []string{"└", "┘"} {
		if strings.Contains(out, absent) {
			t.Fatalf("active tab should have no bottom border, found %q:\n%s", absent, out)
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
	// Both titles are present regardless of which is active.
	for _, out := range []string{active0, active1} {
		for _, title := range titles {
			if !strings.Contains(ansi.Strip(out), title) {
				t.Fatalf("strip missing %q: %q", title, ansi.Strip(out))
			}
		}
	}
	if !strings.Contains(active0, "\x1b[") {
		t.Fatal("active chip should be styled with escape sequences")
	}
}
