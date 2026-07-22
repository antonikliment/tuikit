package tuikit

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPanelStyleFocusedUsesDoubleBorder(t *testing.T) {
	theme := DefaultTheme()
	normal := ansi.Strip(theme.PanelStyle(theme.Cyan, false).Render("x"))
	focused := ansi.Strip(theme.PanelStyle(theme.Cyan, true).Render("x"))

	if !strings.Contains(normal, "┌") {
		t.Fatalf("unfocused panel should use a normal border:\n%s", normal)
	}
	if !strings.Contains(focused, "╔") {
		t.Fatalf("focused panel should use a double border:\n%s", focused)
	}
}

func TestPanelRendersContentAndSize(t *testing.T) {
	theme := DefaultTheme()
	out := Panel{Theme: theme, Accent: theme.Green, Width: 20}.Render("body-text")
	stripped := ansi.Strip(out)
	if !strings.Contains(stripped, "body-text") {
		t.Fatalf("panel missing content:\n%s", stripped)
	}
	for _, line := range strings.Split(stripped, "\n") {
		if w := ansi.StringWidth(line); w != 20 {
			t.Fatalf("line width = %d, want 20: %q", w, line)
		}
	}
}
