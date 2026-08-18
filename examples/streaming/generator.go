package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// The generator produces an endless markdown document one block at a time, so
// the renderer can be watched under a stream that never settles. Text content
// is irrelevant to what is being tested — boundary handling is — so the corpus
// is embedded rather than fetched: it runs offline, and a fixed -seed makes any
// glitch you spot replayable.

var words = strings.Fields(`
lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor
incididunt ut labore et dolore magna aliqua enim ad minim veniam quis nostrud
exercitation ullamco laboris nisi aliquip ex ea commodo consequat duis aute
irure in reprehenderit voluptate velit esse cillum eu fugiat nulla pariatur
excepteur sint occaecat cupidatat non proident sunt culpa qui officia deserunt
mollit anim id est laborum
`)

// snippets are real code, so chroma has something to highlight when a fence
// closes — the moment the component exists to make visible.
var snippets = []struct{ lang, code string }{
	{"go", `func boundary(text string) int {
	for i, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			return i
		}
	}
	return -1
}`},
	{"python", `def stream(chunks, width=80):
    buffer = ""
    for chunk in chunks:
        buffer += chunk
        yield render(buffer, width)`},
	{"sh", `#!/usr/bin/env bash
set -euo pipefail
for f in *.md; do
  printf '%s\n' "$f"
done`},
	{"json", `{
  "seed": 7,
  "blocks": ["paragraph", "list", "fence"],
  "indefinite": true
}`},
}

// generator emits blocks of roughly size units each: paragraphs of size
// sentences, lists of size items, tables of size rows.
type generator struct {
	rng  *rand.Rand
	size int
	n    int
}

func newGenerator(seed uint64, size int) *generator {
	return &generator{rng: rand.New(rand.NewPCG(seed, seed)), size: max(1, size)}
}

// next returns the next block, including the blank line that ends it.
func (g *generator) next() string {
	g.n++
	// Weighted so prose dominates and the awkward constructs still show up
	// often enough to watch.
	switch g.rng.IntN(10) {
	case 0, 1, 2, 3:
		return g.paragraph()
	case 4, 5:
		return g.list()
	case 6:
		return g.fence()
	case 7:
		return g.table()
	case 8:
		return g.quote()
	default:
		return g.heading()
	}
}

func (g *generator) paragraph() string {
	var b strings.Builder
	for range g.size {
		b.WriteString(g.sentence())
		b.WriteString(" ")
	}
	return strings.TrimSpace(b.String()) + "\n\n"
}

func (g *generator) sentence() string {
	n := 6 + g.rng.IntN(12)
	parts := make([]string, n)
	for i := range parts {
		parts[i] = words[g.rng.IntN(len(words))]
	}
	// Sprinkle emphasis so inline styling is exercised too.
	if n > 4 {
		parts[2] = "**" + parts[2] + "**"
		parts[4] = "`" + parts[4] + "`"
	}
	sentence := strings.Join(parts, " ")
	return strings.ToUpper(sentence[:1]) + sentence[1:] + "."
}

func (g *generator) list() string {
	var b strings.Builder
	ordered := g.rng.IntN(2) == 0
	for i := range g.size {
		marker := "-"
		if ordered {
			marker = fmt.Sprintf("%d.", i+1)
		}
		fmt.Fprintf(&b, "%s %s\n", marker, g.phrase(5))
		if g.rng.IntN(4) == 0 {
			fmt.Fprintf(&b, "  - %s\n", g.phrase(4))
		}
	}
	return b.String() + "\n"
}

func (g *generator) fence() string {
	s := snippets[g.rng.IntN(len(snippets))]
	return fmt.Sprintf("```%s\n%s\n```\n\n", s.lang, s.code)
}

func (g *generator) table() string {
	var b strings.Builder
	b.WriteString("| block | bytes | settled |\n|---|---|---|\n")
	for range g.size {
		fmt.Fprintf(&b, "| %s | %d | %v |\n",
			words[g.rng.IntN(len(words))], g.rng.IntN(9000), g.rng.IntN(2) == 0)
	}
	return b.String() + "\n"
}

func (g *generator) quote() string {
	return "> " + g.sentence() + "\n> " + g.sentence() + "\n\n"
}

func (g *generator) heading() string {
	return fmt.Sprintf("## %s\n\n", g.phrase(3))
}

func (g *generator) phrase(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = words[g.rng.IntN(len(words))]
	}
	return strings.Join(parts, " ")
}
