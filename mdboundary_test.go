package tuikit

import (
	"strings"
	"testing"
)

// A settled paragraph followed by a blank line is the base case every other
// test is a refusal of.
func TestBoundaryOfCutsAfterASettledParagraph(t *testing.T) {
	text := "Done streaming this bit.\n\nStill arriving"
	if got, want := boundaryOf(text), len("Done streaming this bit.\n\n"); got != want {
		t.Fatalf("boundaryOf(%q) = %d, want %d", text, got, want)
	}
}

// Each of these can still be changed by text that has not arrived, so no cut is
// safe even though a blank line is present.
func TestBoundaryOfRefusesOpenConstructs(t *testing.T) {
	cases := map[string]string{
		"unclosed fence":   "```go\nfunc main() {\n\nstill inside\n",
		"list item":        "- first item\n\n- second item\n",
		"numbered item":    "1. first item\n\n2. second item\n",
		"link reference":   "[docs]: https://example.com\n\nSee [docs].\n",
		"html block":       "<div>\n\nstill inside\n",
		"setext underline": "A heading maybe\n\n===\n",
		"block quote":      "> quoted line\n\nafter\n",
		"table row":        "| a | b |\n\nafter\n",
		"indented code":    "    indented := true\n\nafter\n",
	}
	for name, text := range cases {
		if got := boundaryOf(text); got != 0 {
			t.Fatalf("%s: boundaryOf(%q) = %d, want 0", name, text, got)
		}
	}
}

// A fence that has closed is settled, which is the case that makes a long code
// block snap into highlighting the moment it ends.
func TestBoundaryOfCutsAfterAClosedFence(t *testing.T) {
	text := "```go\nfunc main() {}\n```\n\nnext paragraph"
	if got, want := boundaryOf(text), len("```go\nfunc main() {}\n```\n\n"); got != want {
		t.Fatalf("boundaryOf = %d, want %d", got, want)
	}
}

// The cut always advances to the last safe boundary, not the first.
func TestBoundaryOfPrefersTheLatestSafeCut(t *testing.T) {
	text := "one\n\ntwo\n\nthree"
	if got, want := boundaryOf(text), len("one\n\ntwo\n\n"); got != want {
		t.Fatalf("boundaryOf = %d, want %d", got, want)
	}
}

// A buffer with nothing settled must render entirely as tail rather than
// reporting a cut past the end of the text.
func TestBoundaryOfHandlesShortAndEmptyBuffers(t *testing.T) {
	for _, text := range []string{"", "no blank line yet", "\n"} {
		if got := boundaryOf(text); got < 0 || got > len(text) {
			t.Fatalf("boundaryOf(%q) = %d, out of range", text, got)
		}
	}
}

// isListItem has to tell a numbered item from a decimal or a bare number, since
// treating ordinary prose as a list would stall the boundary forever.
func TestIsListItemDistinguishesNumbersFromItems(t *testing.T) {
	items := []string{"- a", "* a", "+ a", "1. a", "2) a", "42. a"}
	notItems := []string{"1.5 metres", "-dash", "*bold*", "12", "1.", "a - b"}
	for _, body := range items {
		if !isListItem(body) {
			t.Fatalf("isListItem(%q) = false, want true", body)
		}
	}
	for _, body := range notItems {
		if isListItem(body) {
			t.Fatalf("isListItem(%q) = true, want false", body)
		}
	}
}

// The invariant the whole design rests on: whatever order the bytes arrive in,
// the settled prefix must be a prefix of the source, and it must only grow.
func TestBoundaryOfAdvancesMonotonicallyWhileStreaming(t *testing.T) {
	const doc = "Intro line.\n\n```go\nfunc main() {}\n```\n\nOutro line.\n\ntail"
	previous := 0
	for i := range len(doc) + 1 {
		cut := boundaryOf(doc[:i])
		if cut > i {
			t.Fatalf("at %d bytes: cut %d past end of buffer", i, cut)
		}
		if !strings.HasPrefix(doc, doc[:i][:cut]) {
			t.Fatalf("at %d bytes: cut %d is not a prefix of the source", i, cut)
		}
		if cut < previous {
			t.Fatalf("at %d bytes: cut went backwards, %d then %d", i, previous, cut)
		}
		previous = cut
	}
}
