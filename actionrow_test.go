package tuikit

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestActionRowShowsAllLabels(t *testing.T) {
	theme := DefaultTheme()
	labels := []string{"Start", "Stop", "Restart"}
	out := ansi.Strip(theme.ActionRow(theme.Cyan, 1, labels, true))
	for _, want := range append([]string{"Actions:"}, labels...) {
		if !strings.Contains(out, want) {
			t.Fatalf("action row missing %q: %q", want, out)
		}
	}
}

func TestActionRowBracketsSelectedWhenFocused(t *testing.T) {
	theme := DefaultTheme()
	out := ansi.Strip(theme.ActionRow(theme.Cyan, 1, []string{"Start", "Stop", "Restart"}, true))
	if !strings.Contains(out, "[Stop]") {
		t.Fatalf("focused selected label should be bracketed: %q", out)
	}
	if strings.Contains(out, "[Start]") || strings.Contains(out, "[Restart]") {
		t.Fatalf("only the selected label should be bracketed: %q", out)
	}
}

func TestActionRowUnfocusedIsMutedAndUnbracketed(t *testing.T) {
	theme := DefaultTheme()
	focused := theme.ActionRow(theme.Cyan, 1, []string{"Start", "Stop"}, true)
	unfocused := theme.ActionRow(theme.Cyan, 1, []string{"Start", "Stop"}, false)
	if focused == unfocused {
		t.Fatal("focus should change the rendered row")
	}
	if strings.Contains(ansi.Strip(unfocused), "[") {
		t.Fatalf("unfocused row should not bracket any action: %q", ansi.Strip(unfocused))
	}
}
