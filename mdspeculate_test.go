package tuikit

import (
	"strings"
	"testing"
)

func TestCloseOpen(t *testing.T) {
	cases := []struct {
		name, tail, want string
	}{
		{"nothing open", "plain prose", "plain prose"},
		{"balanced", "a **bold** word", "a **bold** word"},
		{"strong", "a **bold", "a **bold**"},
		{"emphasis", "a *slanted", "a *slanted*"},
		{"underscore", "a _slanted", "a _slanted_"},
		{"strikethrough", "a ~~struck", "a ~~struck~~"},
		{"code span", "call `Render", "call `Render`"},
		{"nested, closed innermost first", "**bold and *both", "**bold and *both***"},
		{"link text", "see [the note", "see [the note]"},
		{"link destination", "see [the note](docs/", "see [the note](docs/)"},

		// A delimiter with nothing after it is the start of a longer one far
		// more often than an abandoned span, so it is dropped rather than closed.
		{"trailing opener dropped", "a *", "a"},
		{"trailing strong dropped", "a **", "a"},
		{"trailing bracket dropped", "see [", "see"},

		// Emphasis does not cross a blank line, and nothing inside a code span
		// or a fence is inline markdown.
		{"blank line abandons", "a *slanted\n\nnew paragraph", "a *slanted\n\nnew paragraph"},
		{"literal inside code span", "`a *b", "`a *b`"},
		{"code span swallows a longer run", "a `x ``` y", "a `x ``` y`"},
		{"only an equal run closes a code span", "a ``x ` y", "a ``x ` y``"},
		{"escaped delimiter", `a \*not emphasis`, `a \*not emphasis`},

		// The fence is the case this approach gets for free: CommonMark closes
		// it at the end of the document, so no synthetic closer is needed and
		// nothing inside it is scanned.
		{"open fence untouched", "```go\nx := *p\n", "```go\nx := *p\n"},
		{"closed fence, then prose", "```\nx\n```\n\na **b", "```\nx\n```\n\na **b**"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := closeOpen(c.tail); got != c.want {
				t.Errorf("closeOpen(%q)\n got %q\nwant %q", c.tail, got, c.want)
			}
		})
	}
}

// A wrong guess must never outlive the frame it was drawn in. Once the tail has
// settled it is rendered from the source, so whatever was speculated about it
// has to be gone.
func TestSpeculativeClosersNeverReachSettledOutput(t *testing.T) {
	var seen []string
	record := func(text string, width int) string {
		seen = append(seen, text)
		return text
	}
	s := NewStreamingMarkdown(record)

	const document = "a **bold** paragraph.\n\nand a second one.\n\n"
	for i := 1; i <= len(document); i++ {
		s.Render(document[:i], 40)
	}
	if got := s.Render(document, 40); strings.Contains(got, "****") {
		t.Errorf("doubled closers in settled output: %q", got)
	}
	for _, text := range seen {
		if strings.Contains(text, "*****") {
			t.Errorf("a closer was rendered into the prefix: %q", text)
		}
	}
}

// The tail is re-rendered every frame it changes, against a wrap in the default
// mode. This is the number to check before turning speculation on over a fast
// stream; it measures the scan alone, not the markdown engine behind it.
func BenchmarkCloseOpen(b *testing.B) {
	tail := strings.Repeat("Some **prose** with `code` and a [link](to/somewhere) in it. ", 12) + "and a **partial"
	b.ReportAllocs()
	for b.Loop() {
		closeOpen(tail)
	}
}

// The property that makes speculation safe to leave on: closing a construct may
// add markers and it may drop a trailing one, but it must never lose or reorder
// the text itself. Checked over every prefix of the corpus, which is where a
// scan that mishandles a partial construct would show up.
func TestCloseOpenPreservesText(t *testing.T) {
	strip := func(s string) string {
		return strings.Map(func(r rune) rune {
			if strings.ContainsRune("*_~`[]()\\ \t\n", r) {
				return -1
			}
			return r
		}, s)
	}
	for name, doc := range corpus() {
		for i := range len(doc) + 1 {
			tail := doc[:i]
			if got, want := strip(closeOpen(tail)), strip(tail); got != want {
				t.Fatalf("%s at %d bytes: text changed\n got %q\nwant %q", name, i, got, want)
			}
		}
	}
}
