package tuikit

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestField(t *testing.T) {
	got := Field("Model", "gemma.gguf")
	if !strings.HasPrefix(got, "Model:") || !strings.HasSuffix(got, "gemma.gguf") {
		t.Fatalf("Field = %q", got)
	}
}

func TestVerticalSlice(t *testing.T) {
	content := "a\nb\nc\nd\ne"
	if got := VerticalSlice(content, 0, 3); got != "a\nb\nc" {
		t.Fatalf("clip = %q", got)
	}
	if got := VerticalSlice(content, 2, 2); got != "c\nd" {
		t.Fatalf("offset clip = %q", got)
	}
	if got := VerticalSlice(content, 0, 0); got != content {
		t.Fatalf("height 0 should be unchanged: %q", got)
	}
	if got := VerticalSlice("a\nb", 0, 5); got != "a\nb" {
		t.Fatalf("short content should be unchanged: %q", got)
	}
}

func TestFlowWraps(t *testing.T) {
	blocks := []string{"aaa", "bbb", "ccc", "ddd"}
	out := Flow(7, 1, blocks) // fits ~2 per row at width 7
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapping into multiple rows, got %d:\n%s", len(lines), out)
	}
	for _, b := range blocks {
		if !strings.Contains(out, b) {
			t.Fatalf("flow dropped %q:\n%s", b, out)
		}
	}
}

func TestFlowEmptyWidth(t *testing.T) {
	if got := Flow(0, 1, []string{"x"}); got != "" {
		t.Fatalf("Flow(0,...) = %q, want empty", got)
	}
}

func TestRuleWidth(t *testing.T) {
	theme := DefaultTheme()
	if got := ansi.StringWidth(ansi.Strip(theme.Rule(30))); got != 24 {
		t.Fatalf("rule width = %d, want 24 (width-6)", got)
	}
}

func TestStatusTitle(t *testing.T) {
	theme := DefaultTheme()
	out := ansi.Strip(theme.StatusTitle("chat", "Running", theme.Cyan, theme.Green, 60))
	if !strings.Contains(out, "chat") || !strings.Contains(out, "Running") {
		t.Fatalf("StatusTitle = %q", out)
	}
}
