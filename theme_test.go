package tuikit

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestDefaultThemeColorsSet(t *testing.T) {
	theme := DefaultTheme()
	colors := map[string]any{
		"Green": theme.Green, "Blue": theme.Blue, "Yellow": theme.Yellow,
		"Red": theme.Red, "Cyan": theme.Cyan, "Muted": theme.Muted,
		"Brand": theme.Brand, "TabActiveFg": theme.TabActiveFg, "FocusBorder": theme.FocusBorder,
	}
	for name, c := range colors {
		if c == nil {
			t.Fatalf("DefaultTheme().%s is nil", name)
		}
	}
}

func TestThemeStyleHelpersRenderText(t *testing.T) {
	theme := DefaultTheme()
	cases := map[string]string{
		"muted":  theme.MutedStyle().Render("muted"),
		"subtle": theme.SubtleStyle().Render("subtle"),
		"brand":  theme.BrandStyle().Render("brand"),
		"accent": theme.Accent(theme.Cyan).Render("accent"),
	}
	for want, rendered := range cases {
		if got := ansi.Strip(rendered); got != want {
			t.Fatalf("%s style text = %q, want %q", want, got, want)
		}
	}
}
