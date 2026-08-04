package tuikit_test

import (
	"bytes"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

var red = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

func TestPlainLeavesTextAlone(t *testing.T) {
	if got := tuikit.Plain(red, "boom"); got != "boom" {
		t.Errorf("Plain = %q, want %q", got, "boom")
	}
}

func TestPainterForNonTerminalIsPlain(t *testing.T) {
	var buf bytes.Buffer
	if got := tuikit.PainterFor(&buf)(red, "boom"); got != "boom" {
		t.Errorf("PainterFor(buffer) painted %q, want plain text", got)
	}
}

func TestColorEnabledHonoursNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if tuikit.ColorEnabled(&bytes.Buffer{}) {
		t.Error("ColorEnabled = true with NO_COLOR set")
	}
}

func TestIsTerminalWriterOnBuffer(t *testing.T) {
	if tuikit.IsTerminalWriter(&bytes.Buffer{}) {
		t.Error("IsTerminalWriter(buffer) = true, want false")
	}
}

func TestStatusWord(t *testing.T) {
	theme := tuikit.DefaultTheme()
	for _, tc := range []struct {
		word  string
		level tuikit.Level
	}{
		{"running", tuikit.LevelSuccess},
		{"OK", tuikit.LevelSuccess},
		{"download_failed", tuikit.LevelError},
		{"errored", tuikit.LevelError},
		{"pending", tuikit.LevelWarning},
		{"", tuikit.LevelInfo},
	} {
		if got := tuikit.ClassifyStatus(tc.word); got != tc.level {
			t.Errorf("ClassifyStatus(%q) = %v, want %v", tc.word, got, tc.level)
		}
	}
	// Painted output keeps the word itself intact.
	if got := theme.StatusWord(tuikit.Paint, "running"); !strings.Contains(got, "running") {
		t.Errorf("StatusWord dropped its word: %q", got)
	}
	if got := theme.StatusWord(tuikit.Plain, ""); got != "" {
		t.Errorf("StatusWord(\"\") = %q, want empty", got)
	}
}

func TestTitleize(t *testing.T) {
	for input, want := range map[string]string{
		"running_profiles": "Running profiles",
		"runningProfiles":  "Running profiles",
		"ok":               "Ok",
		"":                 "",
	} {
		if got := tuikit.Titleize(input); got != want {
			t.Errorf("Titleize(%q) = %q, want %q", input, got, want)
		}
	}
}
