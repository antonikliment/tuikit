package tuikit

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestTruncMiddle(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"fits", "abc", 5, "abc"},
		{"exact", "abcde", 5, "abcde"},
		{"elides", "abcdefghij", 5, "ab…ij"},
		{"width one", "abcdef", 1, "abcdef"},
		{"width zero", "abcdef", 0, "abcdef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TruncMiddle(c.in, c.width); got != c.want {
				t.Fatalf("TruncMiddle(%q,%d) = %q, want %q", c.in, c.width, got, c.want)
			}
			if w := ansi.StringWidth(TruncMiddle(c.in, c.width)); c.width > 1 && len([]rune(c.in)) > c.width && w > c.width {
				t.Fatalf("elided width %d exceeds %d", w, c.width)
			}
		})
	}
	// multibyte runes are not split
	if got := TruncMiddle("héllo wörld", 5); ansi.StringWidth(got) > 5 {
		t.Fatalf("multibyte truncation too wide: %q", got)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{3 * 1024 * 1024 * 1024, "3.0 GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TiB"},
		{2 * 1024 * 1024 * 1024 * 1024 * 1024, "2.0 PiB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Fatalf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAdaptiveWidth(t *testing.T) {
	// Narrow total falls back to a single column at least min wide.
	if got := AdaptiveWidth(30, 2, 28, 48); got < 28 {
		t.Fatalf("single column width %d < min 28", got)
	}
	// Result never exceeds max.
	if got := AdaptiveWidth(500, 2, 28, 48); got > 48 {
		t.Fatalf("width %d > max 48", got)
	}
	// Result never drops below min.
	for total := 1; total < 200; total += 7 {
		if got := AdaptiveWidth(total, 2, 28, 48); got < 28 {
			t.Fatalf("AdaptiveWidth(%d,...) = %d < min", total, got)
		}
	}
}

func TestStatusConfirmFlow(t *testing.T) {
	fired := false
	fire := func() tea.Cmd { fired = true; return nil }

	var s Status
	// First press arms, does not fire.
	if cmd := s.Confirm("target-a", false, fire); cmd != nil || fired {
		t.Fatalf("first press should arm, not fire")
	}
	if s.Pending() != "target-a" {
		t.Fatalf("pending = %q, want target-a", s.Pending())
	}
	// A confirm-key press on a different target does nothing.
	if cmd := s.Confirm("target-b", true, fire); cmd != nil || fired {
		t.Fatalf("confirm on unarmed target should be a no-op")
	}
	// Second press of the armed target fires and disarms.
	s.Confirm("target-a", false, fire)
	if !fired || s.Pending() != "" {
		t.Fatalf("second press should fire and disarm; fired=%v pending=%q", fired, s.Pending())
	}
}

func TestStatusSetResultAndRows(t *testing.T) {
	theme := DefaultTheme()
	var s Status
	s.Confirm("x", false, func() tea.Cmd { return nil })

	s.SetResult(nil, "done")
	if s.Pending() != "" {
		t.Fatalf("SetResult should clear pending")
	}
	rows := s.AppendRows(theme, nil)
	if len(rows) != 1 || !strings.Contains(ansi.Strip(rows[0]), "done") {
		t.Fatalf("success row = %v", rows)
	}

	s.SetResult(errTest{}, "ignored")
	rows = s.AppendRows(theme, nil)
	if len(rows) != 1 || !strings.Contains(ansi.Strip(rows[0]), "boom") {
		t.Fatalf("error row = %v", rows)
	}
}

type errTest struct{}

func (errTest) Error() string { return "boom" }
