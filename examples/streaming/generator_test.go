package main

import (
	"strings"
	"testing"

	"github.com/antonikliment/tuikit"
	"github.com/antonikliment/tuikit/markdown"
)

// The generator picks a block type at random, so a rare branch could produce
// something that breaks the renderer long after the demo starts. Drive a lot of
// blocks through the real pipeline and assert it stays sane.
func TestGeneratedStreamRendersWithoutLosingText(t *testing.T) {
	streamer := tuikit.NewStreamingMarkdown(markdown.New(tuikit.DefaultTheme()))
	g := newGenerator(7, 3)

	buffer, settled := "", 0
	for range 60 {
		buffer += g.next()
		out := streamer.Render(buffer, 60)
		if out == "" {
			t.Fatalf("empty render at %d bytes", len(buffer))
		}
		if got := streamer.Settled(); got < settled {
			t.Fatalf("settled went backwards at %d bytes: %d then %d", len(buffer), settled, got)
		} else if got > len(buffer) {
			t.Fatalf("settled %d past buffer end %d", got, len(buffer))
		} else {
			settled = got
		}
	}
	if settled == 0 {
		t.Fatal("nothing ever settled across 60 blocks")
	}
	if strings.TrimSpace(buffer[settled:]) == "" && settled != len(buffer) {
		t.Fatalf("tail is blank but settled %d != buffer %d", settled, len(buffer))
	}
}

// A fixed seed has to replay identically, or a glitch spotted on screen cannot
// be reproduced.
func TestGeneratorIsReproducibleForASeed(t *testing.T) {
	first, second := newGenerator(7, 3), newGenerator(7, 3)
	for i := range 50 {
		if a, b := first.next(), second.next(); a != b {
			t.Fatalf("block %d differs between runs of seed 7:\n%q\n%q", i, a, b)
		}
	}
	if a, b := newGenerator(7, 3).next(), newGenerator(8, 3).next(); a == b {
		t.Fatalf("seeds 7 and 8 produced the same first block %q", a)
	}
}
