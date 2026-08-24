package tuikit

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestSegmentBar(t *testing.T) {
	style := lipgloss.NewStyle()
	out := ansi.Strip(SegmentBar(20, []Segment{
		{Label: "prompt", Value: "400", Share: 0.4, Style: style},
		{Label: "tiny", Value: "1", Share: 0.001, Style: style},
	}, Segment{Label: "free", Value: "599", Share: 0.599, Style: style}))

	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected bar + 3 legend rows, got %d lines:\n%s", len(lines), out)
	}
	if got := strings.Count(lines[0], "█") + strings.Count(lines[0], "░"); got != 20 {
		t.Fatalf("bar is %d cells wide, want 20:\n%s", got, out)
	}
	// A positive share always paints at least one cell: 8 prompt + 1 tiny.
	if got := strings.Count(lines[0], "█"); got != 9 {
		t.Fatalf("filled cells = %d, want 9:\n%s", got, out)
	}
	for _, want := range []string{"■ prompt", "400", "40.0%", "■ tiny", "0.1%", "□ free", "59.9%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("legend missing %q:\n%s", want, out)
		}
	}
}

func TestSegmentBarClampsOverflow(t *testing.T) {
	style := lipgloss.NewStyle()
	out := ansi.Strip(SegmentBar(10, []Segment{
		{Label: "a", Value: "9", Share: 0.9, Style: style},
		{Label: "b", Value: "9", Share: 0.9, Style: style},
	}, Segment{Label: "free", Value: "0", Share: 0, Style: style}))
	bar := strings.Split(out, "\n")[0]
	if got := strings.Count(bar, "█") + strings.Count(bar, "░"); got != 10 {
		t.Fatalf("overflowing shares must clamp to width, got %d cells:\n%s", got, out)
	}
}
