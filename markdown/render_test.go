package markdown

import (
	"strings"
	"testing"

	"github.com/antonikliment/tuikit"

	"github.com/charmbracelet/x/ansi"
)

// Rendered output carries ANSI styling, so assertions compare the visible text.
func visible(s string) string { return strings.TrimSpace(ansi.Strip(s)) }

func TestNewRendersMarkdownStructure(t *testing.T) {
	render := New(tuikit.DefaultTheme())
	got := visible(render("# Title\n\nSome **bold** text.", 40))
	if !strings.Contains(got, "TITLE") {
		t.Fatalf("render = %q, want an uppercased H1", got)
	}
	if !strings.Contains(got, "bold") {
		t.Fatalf("render = %q, want the body text", got)
	}
	if strings.Contains(got, "**") {
		t.Fatalf("render = %q, want the emphasis markers consumed", got)
	}
}

// The host composes its own spacing, so glamour's line padding and bracketing
// blank lines must not survive.
func TestNewTrimsGlamourPadding(t *testing.T) {
	got := New(tuikit.DefaultTheme())("Short line.", 60)
	for _, line := range strings.Split(got, "\n") {
		if strings.HasSuffix(ansi.Strip(line), " ") {
			t.Fatalf("line %q keeps trailing padding", ansi.Strip(line))
		}
	}
	if strings.HasPrefix(got, "\n") || strings.HasSuffix(got, "\n") {
		t.Fatalf("render = %q, want no bracketing blank lines", got)
	}
}

func TestNewReturnsInputForEmptyOrZeroWidth(t *testing.T) {
	render := New(tuikit.DefaultTheme())
	if got := render("", 40); got != "" {
		t.Fatalf("render of empty text = %q, want %q", got, "")
	}
	if got := render("text", 0); got != "text" {
		t.Fatalf("render at width 0 = %q, want %q", got, "text")
	}
}

// The pairing this package exists for: streamed blocks come out formatted while
// the unfinished tail stays legible.
func TestNewDrivesStreamingMarkdown(t *testing.T) {
	s := tuikit.NewStreamingMarkdown(New(tuikit.DefaultTheme()))
	got := visible(s.Render("# Title\n\nA paragraph.\n\n```go\nfunc ma", 60))
	if !strings.Contains(got, "TITLE") {
		t.Fatalf("streamed = %q, want the settled heading formatted", got)
	}
	if !strings.Contains(got, "func ma") {
		t.Fatalf("streamed = %q, want the partial fence shown raw", got)
	}
}

// One renderer is built per width and reused; a resize builds another. Failing
// this means rebuilding a goldmark pipeline every frame.
func TestNewReusesRenderersPerWidth(t *testing.T) {
	render := New(tuikit.DefaultTheme())
	first := render("A line of text that will wrap differently.", 20)
	second := render("A line of text that will wrap differently.", 20)
	if first != second {
		t.Fatalf("same width rendered differently:\n%q\n%q", first, second)
	}
	if wide := render("A line of text that will wrap differently.", 70); wide == first {
		t.Fatalf("width 70 rendered identically to width 20, want a different wrap")
	}
}
