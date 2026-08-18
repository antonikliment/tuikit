package tuikit

import (
	"fmt"
	"strings"
	"testing"
)

// upper is a stand-in markdown renderer: it is obviously distinguishable from
// raw text in assertions, and it lets these tests run without a markdown engine
// (the root package deliberately has none).
func upper(text string, width int) string {
	return fmt.Sprintf("[%d]%s", width, strings.ToUpper(strings.TrimSpace(text)))
}

// counting wraps a RenderFunc to record how often it actually ran, which is how
// the cache is observed — the cache is otherwise invisible from outside.
func counting(render RenderFunc, calls *int) RenderFunc {
	return func(text string, width int) string {
		*calls++
		return render(text, width)
	}
}

func TestRenderFormatsSettledBlocksAndLeavesTheTailRaw(t *testing.T) {
	s := NewStreamingMarkdown(upper, WithRawTail())
	got := s.Render("settled block\n\nstill arriving", 40)
	want := "[40]SETTLED BLOCK\n\nstill arriving"
	if got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

// Nothing has settled yet, so the whole buffer is tail and the renderer must
// not be called with a partial block at all.
func TestRenderLeavesAnUnsettledBufferEntirelyRaw(t *testing.T) {
	calls := 0
	s := NewStreamingMarkdown(counting(upper, &calls), WithRawTail())
	if got, want := s.Render("- half a list item", 40), "- half a list item"; got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
	if calls != 0 {
		t.Fatalf("renderer called %d times on an unsettled buffer, want 0", calls)
	}
}

// The point of the cache: a stream grows by a character a frame, and the
// settled prefix must not be re-rendered for every one of them.
func TestRenderReusesTheCacheWhileOnlyTheTailGrows(t *testing.T) {
	calls := 0
	s := NewStreamingMarkdown(counting(upper, &calls), WithRawTail())
	for _, tail := range []string{"a", "ab", "abc", "abcd"} {
		s.Render("settled\n\n"+tail, 40)
	}
	if calls != 1 {
		t.Fatalf("renderer called %d times, want 1", calls)
	}
}

func TestRenderRebuildsTheCacheWhenTheWidthChanges(t *testing.T) {
	calls := 0
	s := NewStreamingMarkdown(counting(upper, &calls), WithRawTail())
	s.Render("settled\n\ntail", 40)
	got := s.Render("settled\n\ntail", 80)
	if calls != 2 {
		t.Fatalf("renderer called %d times across a resize, want 2", calls)
	}
	if !strings.HasPrefix(got, "[80]") {
		t.Fatalf("Render after resize = %q, want it laid out at width 80", got)
	}
}

// A retried turn or an edited prompt replaces the buffer rather than extending
// it. Reusing the previous message's output there would show one answer's text
// under another's, so a changed prefix must miss even at the same length.
func TestRenderRebuildsTheCacheWhenTheBufferIsRewritten(t *testing.T) {
	s := NewStreamingMarkdown(upper)
	s.Render("initial\n\ntail", 40)
	got := s.Render("REPLACED\n\ntail", 40)
	if !strings.HasPrefix(got, "[40]REPLACED") {
		t.Fatalf("Render after a rewrite = %q, want the new text", got)
	}
}

// A long fence shows as plain text while it streams and becomes formatted the
// moment it closes — the behavior the component exists for.
func TestRenderFormatsACodeFenceOnceItCloses(t *testing.T) {
	s := NewStreamingMarkdown(upper, WithRawTail())
	open := "intro\n\n```go\nfunc main() {"
	if got := s.Render(open, 40); !strings.Contains(got, "```go") {
		t.Fatalf("open fence = %q, want the raw fence still visible", got)
	}
	closed := open + "}\n```\n\n"
	if got := s.Render(closed, 40); strings.Contains(got, "```go") {
		t.Fatalf("closed fence = %q, want it formatted rather than raw", got)
	}
}

func TestRenderReturnsNothingAtANonPositiveWidth(t *testing.T) {
	s := NewStreamingMarkdown(upper)
	for _, width := range []int{0, -1} {
		if got := s.Render("settled\n\ntail", width); got != "" {
			t.Fatalf("Render at width %d = %q, want %q", width, got, "")
		}
	}
}

// The invariant borrowed from glow's --flow proposal, in the form that holds
// for a stream with no end: whatever the chunking, every byte of the buffer is
// accounted for exactly once — formatted in the settled prefix or verbatim in
// the tail, never dropped and never shown twice.
func TestRenderAccountsForEveryByteAtEveryChunkSize(t *testing.T) {
	for name, doc := range corpus() {
		for _, chunk := range []int{1, 3, 7, 64} {
			settled := ""
			s := NewStreamingMarkdown(func(text string, width int) string {
				settled = text
				return upper(text, width)
			}, WithRawTail())
			for i := chunk; i <= len(doc)+chunk; i += chunk {
				buffer := doc[:min(i, len(doc))]
				got := s.Render(buffer, 40)
				if !strings.HasPrefix(buffer, settled) {
					t.Fatalf("%s/%d: settled %q is not a prefix of %q", name, chunk, settled, buffer)
				}
				tail := strings.TrimSpace(buffer[len(settled):])
				if tail != "" && !strings.Contains(got, tail) {
					t.Fatalf("%s/%d: tail %q missing from output %q", name, chunk, tail, got)
				}
			}
		}
	}
}

// A document whose last block is closed settles completely, so the streamed
// result matches a single render of the whole text. Documents ending mid-list
// or mid-fence deliberately do not: with no end-of-stream signal there is
// nothing to prove the block finished, and the caller renders directly once the
// turn ends.
func TestRenderFullySettlesADocumentThatEndsOnAClosedBlock(t *testing.T) {
	docs := map[string]string{
		"paragraphs":  "First para.\n\nSecond para.\n\nThird para.\n\n",
		"fenced code": "Intro.\n\n```go\nfunc main() {}\n```\n\n",
		"quote":       "Intro.\n\n> quoted\n> lines\n\nOutro.\n\n",
		"heading":     "# Title\n\nBody text.\n\n",
	}
	for name, doc := range docs {
		s := NewStreamingMarkdown(upper)
		var got string
		for i := 1; i <= len(doc); i++ {
			got = s.Render(doc[:i], 40)
		}
		if want := upper(doc, 40); got != want {
			t.Fatalf("%s:\n got %q\nwant %q", name, got, want)
		}
	}
}

// corpus is the shared set of documents the streaming assertions run over: one
// per markdown construct whose boundaries are non-obvious.
func corpus() map[string]string {
	return map[string]string{
		"paragraphs":  "First para.\n\nSecond para.\n\nThird para.\n",
		"fenced code": "Intro.\n\n```go\nfunc main() {}\n```\n\nOutro.\n",
		"list":        "Intro.\n\n- one\n- two\n- three\n\nOutro.\n",
		"nested list": "Intro.\n\n- one\n  - nested\n- two\n\nOutro.\n",
		"table":       "Intro.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nOutro.\n",
		"quote":       "Intro.\n\n> quoted\n> lines\n\nOutro.\n",
		"setext":      "Intro.\n\nA Heading\n=========\n\nOutro.\n",
		"link refs":   "See [docs].\n\n[docs]: https://example.com\n",
		"html":        "Intro.\n\n<div>raw</div>\n\nOutro.\n",
		"mixed":       "# Title\n\nText.\n\n```sh\nls -la\n```\n\n- a\n- b\n\nEnd.\n",
	}
}

// Settled has to describe the buffer that was actually rendered, since a debug
// marker drawn at the wrong offset is worse than none.
func TestSettledReportsTheRenderedPrefixLength(t *testing.T) {
	s := NewStreamingMarkdown(upper)
	if got := s.Settled(); got != 0 {
		t.Fatalf("Settled before any render = %d, want 0", got)
	}
	text := "settled block\n\nstill arriving"
	s.Render(text, 40)
	if got, want := s.Settled(), len("settled block\n\n"); got != want {
		t.Fatalf("Settled = %d, want %d", got, want)
	}
	if !strings.HasPrefix(text, text[:s.Settled()]) {
		t.Fatalf("Settled %d does not describe a prefix of the buffer", s.Settled())
	}
}
