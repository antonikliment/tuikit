package markdown

import (
	"strings"
	"testing"

	"github.com/antonikliment/tuikit"

	"charm.land/glamour/v2/ansi"
	xansi "github.com/charmbracelet/x/ansi"
)

// Rendered output carries ANSI styling, so assertions compare the visible text.
func visible(s string) string { return strings.TrimSpace(xansi.Strip(s)) }

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
		if strings.HasSuffix(xansi.Strip(line), " ") {
			t.Fatalf("line %q keeps trailing padding", xansi.Strip(line))
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

// Fenced code must actually be highlighted: chroma only runs when the style
// carries a theme name, and forgetting it renders code as plain text with no
// error to notice.
func TestNewHighlightsFencedCode(t *testing.T) {
	got := New(tuikit.DefaultTheme())("```go\nfunc main() { return }\n```", 60)
	if visible(got) == got {
		t.Fatal("code block carries no ANSI styling at all")
	}
	// Distinct tokens must not all share one color, which is what a missing
	// chroma theme looks like: styled as a block, unhighlighted within.
	colors := map[string]bool{}
	for _, token := range []string{"func", "main", "return"} {
		if i := strings.Index(got, token); i > 0 {
			colors[styleBefore(got[:i])] = true
		}
	}
	if len(colors) < 2 {
		t.Fatalf("keywords share one color %v, want chroma highlighting", colors)
	}
}

// styleBefore returns the last ANSI escape sequence in s, i.e. the style in
// force at the end of it.
func styleBefore(s string) string {
	last := strings.LastIndex(s, "\x1b[")
	if last < 0 {
		return ""
	}
	if end := strings.Index(s[last:], "m"); end > 0 {
		return s[last : last+end+1]
	}
	return ""
}

func TestWithSyntaxThemeChangesHighlighting(t *testing.T) {
	code := "```go\nfunc main() { return }\n```"
	dark := New(tuikit.DefaultTheme(), WithSyntaxTheme("tokyonight-night"))(code, 60)
	light := New(tuikit.DefaultTheme(), WithSyntaxTheme("tokyonight-day"))(code, 60)
	if dark == light {
		t.Fatal("dark and light syntax themes rendered identically")
	}
	if visible(dark) != visible(light) {
		t.Fatalf("syntax theme changed the text, not just its color:\n%q\n%q", visible(dark), visible(light))
	}
}

// The two things the speculative tail claims, checked against the real
// engine rather than against the closer-appending scan, which cannot know what
// goldmark and chroma will do with its output.
func TestSpeculativeTailRendersWhatTheRawTailShows(t *testing.T) {
	render := New(tuikit.DefaultTheme())
	raw := tuikit.NewStreamingMarkdown(render, tuikit.WithRawTail())
	spec := tuikit.NewStreamingMarkdown(render)

	// No raw markdown: an unfinished emphasis arrives styled, not as asterisks.
	const emphasis = "Settled prose.\n\nA paragraph with **bo"
	if got := raw.Render(emphasis, 50); !strings.Contains(got, "**bo") {
		t.Errorf("the raw tail should show its markers: %q", got)
	}
	got := spec.Render(emphasis, 50)
	if strings.Contains(got, "**") {
		t.Errorf("markers left in the speculative tail: %q", got)
	}
	if !strings.HasSuffix(visible(got), "bo") {
		t.Errorf("the text itself changed: %q", visible(got))
	}

	// The fence, which is the case the display clock cannot stream at all: an
	// open fence needs no synthetic closer, because CommonMark ends it at the
	// end of the document, so chroma lexes the partial body and it arrives
	// highlighted rather than as plain text.
	const fence = "Settled prose.\n\n```go\nfunc main() {\n"
	rawFence := tuikit.NewStreamingMarkdown(render, tuikit.WithRawTail())
	specFence := tuikit.NewStreamingMarkdown(render)
	if got := rawFence.Render(fence, 50); strings.Contains(got, "\x1b[38") {
		t.Errorf("the raw tail should not be highlighted: %q", got)
	}
	if got := specFence.Render(fence, 50); !strings.Contains(got, "\x1b[38") {
		t.Errorf("the partial fence arrived unhighlighted: %q", got)
	}
}

// What speculation costs: the tail is re-rendered through glamour on every
// frame it changes, where [tuikit.WithRawTail] only wraps it. The settled prefix is
// cached in both, so this is the per-frame difference and nothing else.
func BenchmarkRenderTail(b *testing.B) {
	render := New(tuikit.DefaultTheme())
	tail := strings.Repeat("Some **prose** with `code` in it. ", 6) + "and a **partial"
	grow := func(i int) string { return "Settled prose.\n\n" + tail + strings.Repeat("x", i%7) }

	b.Run("wrap", func(b *testing.B) {
		s := tuikit.NewStreamingMarkdown(render, tuikit.WithRawTail())
		for i := 0; b.Loop(); i++ {
			s.Render(grow(i), 80)
		}
	})
	b.Run("speculative", func(b *testing.B) {
		s := tuikit.NewStreamingMarkdown(render)
		for i := 0; b.Loop(); i++ {
			s.Render(grow(i), 80)
		}
	})
}

// The bug this validation exists for. A typo in a stylesheet name — "tokyo-night"
// for "tokyonight-night" — used to reach chroma, which resolves an unknown name
// to "swapoff": a style that colors six token types and leaves identifiers,
// operators and function names unstyled. Code rendered as near-uniform gray, and
// nothing reported a problem.
func TestWithSyntaxThemeFallsBackOnUnknownName(t *testing.T) {
	code := "```go\nfunc main() { return }\n```"
	want := New(tuikit.DefaultTheme())(code, 60)
	for name, unknown := range map[string]string{
		"a plausible typo": "tokyo-night",
		"nonsense":         "no-such-style-exists",
		"empty":            "",
	} {
		t.Run(name, func(t *testing.T) {
			if got := New(tuikit.DefaultTheme(), WithSyntaxTheme(unknown))(code, 60); got != want {
				t.Fatalf("WithSyntaxTheme(%q) did not fall back to DefaultSyntaxTheme", unknown)
			}
		})
	}
}

// Guards the fallback against quietly becoming chroma's own, which is what the
// gray looked like.
func TestWithSyntaxThemeFallbackIsNotChromas(t *testing.T) {
	code := "```go\nfunc main() { return }\n```"
	got := New(tuikit.DefaultTheme(), WithSyntaxTheme("no-such-style-exists"))(code, 60)
	if swapoff := New(tuikit.DefaultTheme(), WithSyntaxTheme("swapoff"))(code, 60); got == swapoff {
		t.Fatal("an unknown name still resolves to chroma's swapoff fallback")
	}
}

// The hook exists so a host can set what a tuikit.Theme has no opinion about
// without forking styleFor.
func TestWithStyleFuncOverridesThemeMapping(t *testing.T) {
	render := New(tuikit.DefaultTheme(), WithStyleFunc(func(s *ansi.StyleConfig) {
		s.Document.Color = ptr("#ff00ff")
	}))
	got := render("Plain prose.", 40)
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("render = %q, want the hook's color applied", got)
	}
	if visible(got) != "Plain prose." {
		t.Fatalf("the hook changed the text: %q", visible(got))
	}
}

// Ordering is the whole point: the hook runs after the theme mapping, so it wins.
func TestWithStyleFuncRunsAfterThemeMapping(t *testing.T) {
	code := "`inline`"
	themed := New(tuikit.DefaultTheme())(code, 40)
	hooked := New(tuikit.DefaultTheme(), WithStyleFunc(func(s *ansi.StyleConfig) {
		s.Code.Color = ptr("#ff00ff")
	}))(code, 40)
	if themed == hooked {
		t.Fatal("the hook did not override the theme's inline-code color")
	}
}
