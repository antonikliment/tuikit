package main

import (
	"strings"
	"testing"
)

func TestClassifyReportsWhatTheTailIsInside(t *testing.T) {
	cases := map[string]struct {
		tail string
		want openKind
	}{
		"empty":                {"", openNothing},
		"after a blank line":   {"a closed paragraph.\n\n", openNothing},
		"prose":                {"a paragraph still being", openParagraph},
		"prose over two lines": {"a paragraph\nstill being", openParagraph},
		"list":                 {"- an item\n- another", openContainer},
		"numbered list":        {"1. an item\n2. another", openContainer},
		"quote":                {"> quoted", openContainer},
		"table":                {"| a | b |\n|---|---|", openContainer},
		"indented code":        {"    indented := true", openContainer},
		"html":                 {"<div>", openContainer},
		"open fence":           {"```go\nfunc main() {", openFence},
		"tilde fence":          {"~~~\nplain text", openFence},
		"closed fence":         {"```go\nfunc main() {}\n```\n", openNothing},
		"lazy continuation":    {"- an item\nwrapped onto the next line", openContainer},
	}
	for name, c := range cases {
		if got := classify(c.tail).kind; got != c.want {
			t.Errorf("%s: classify(%q) = %v, want %v", name, c.tail, got, c.want)
		}
	}
}

// A fence closes only on its own marker at its own length. Getting this wrong
// in either direction is expensive: a fence read as closed early makes the
// policy stop conserving right when it should start, and one never read as
// closed makes it conserve forever.
func TestClassifyMatchesFenceMarkers(t *testing.T) {
	cases := map[string]struct {
		tail string
		want openKind
	}{
		"tilde does not close backtick": {"```go\ncode\n~~~\n", openFence},
		"backtick does not close tilde": {"~~~\ncode\n```\n", openFence},
		"short run does not close long": {"````\ncode\n```\n", openFence},
		"long run closes short":         {"```\ncode\n````\n", openNothing},
		"info string does not close":    {"```go\ncode\n```python\n", openFence},
		"trailing spaces still close":   {"```go\ncode\n```   \n", openNothing},
	}
	for name, c := range cases {
		if got := classify(c.tail).kind; got != c.want {
			t.Errorf("%s: classify(%q) = %v, want %v", name, c.tail, got, c.want)
		}
	}
}

// A blank line inside a fence is code, not a block separator. Treating it as
// one would report an open fence as settled halfway through.
func TestClassifyIgnoresBlankLinesInsideAFence(t *testing.T) {
	if got := classify("```go\nfunc main() {\n\n}\n").kind; got != openFence {
		t.Fatalf("classify = %v, want the fence still open across a blank line", got)
	}
}

// The size is what separates an ordinary wait from a frozen display, so it has
// to count from where the construct opened, not from the start of the tail.
func TestClassifyMeasuresHowLongTheConstructHasBeenOpen(t *testing.T) {
	const tail = "a closed paragraph.\n\n```go\nfunc main() {"
	fence := "```go\nfunc main() {"

	got := classify(tail)
	if got.kind != openFence {
		t.Fatalf("classify = %v, want %v", got.kind, openFence)
	}
	if want := len(fence); got.open != want {
		t.Fatalf("open = %d bytes, want %d — measured from the opening line", got.open, want)
	}
}

// A construct only grows while it stays open, and the count resets when the
// next one starts. The policy reads this every tick, so a drift here would
// quietly bias the pacing.
func TestClassifyGrowsWhileOpenAndResetsAfter(t *testing.T) {
	const doc = "```go\nfunc main() {}\n```\n\nnext paragraph"
	previous := 0
	for i := range len(doc) + 1 {
		got := classify(doc[:i])
		if got.kind == openFence && got.open < previous {
			t.Fatalf("at %d bytes: fence shrank from %d to %d", i, previous, got.open)
		}
		if got.kind == openFence {
			previous = got.open
		} else {
			previous = 0
		}
		if got.open > i {
			t.Fatalf("at %d bytes: open = %d, past the start of the tail", i, got.open)
		}
	}
	if got := classify(doc); got.kind != openParagraph {
		t.Fatalf("classify at end = %v, want the fence closed and a paragraph open", got.kind)
	}
}

// The generator is the corpus the demo runs on, so the classifier has to agree
// with it: every block it emits ends closed, and every fence it emits is seen
// as open until its final line arrives.
func TestClassifyAgreesWithTheGenerator(t *testing.T) {
	gen := newGenerator(7, 4)
	for range 200 {
		block := gen.next()
		if got := classify(block).kind; got != openNothing {
			t.Fatalf("classify(%q) = %v, want a completed block to close", block, got)
		}
		if !strings.HasPrefix(block, "```") {
			continue
		}
		partial := strings.TrimSuffix(block, "```\n\n")
		if got := classify(partial).kind; got != openFence {
			t.Fatalf("classify(%q) = %v, want the fence open before its closing marker", partial, got)
		}
	}
}
