package tuikit

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestHelpLineRendersKeysAndDescriptions(t *testing.T) {
	up := key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "Navigate"))
	quit := key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("Ctrl+C", "Quit"))
	out := ansi.Strip(HelpLine(up, quit))
	for _, want := range []string{"↑/↓", "Navigate", "Ctrl+C", "Quit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help line missing %q: %q", want, out)
		}
	}
}

func TestHelpBrightensKeyAndDescColors(t *testing.T) {
	h := Help()
	if got := h.Styles.ShortKey.GetForeground(); got != lipgloss.Color("252") {
		t.Fatalf("ShortKey foreground = %v, want bright key color 252", got)
	}
	if got := h.Styles.ShortDesc.GetForeground(); got != lipgloss.Color("250") {
		t.Fatalf("ShortDesc foreground = %v, want bright desc color 250", got)
	}
}
