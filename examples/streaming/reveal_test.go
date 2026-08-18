package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/antonikliment/tuikit"
	"github.com/antonikliment/tuikit/markdown"
)

// plain is a stand-in renderer: it marks its output so a test can tell rendered
// output from raw source at a glance, and it needs no markdown engine.
func plain(text string, width int) string {
	return fmt.Sprintf("<%d>%s", width, strings.TrimSpace(text))
}

// Nothing settled means nothing to play, and — the point of the mode — nothing
// unfinished is drawn in the meantime.
func TestRevealDrawsNothingUntilABlockSettles(t *testing.T) {
	r := newReveal(plain)
	r.Feed("- half a list item", 40)
	r.Advance(1000)
	if got := r.View(); got != "" {
		t.Fatalf("View = %q, want empty while nothing has settled", got)
	}
}

// The head plays out a settled block a cell at a time, and what is on screen is
// always a prefix of what the block will look like when it finishes.
func TestRevealPlaysASettledBlockOutAPrefixAtATime(t *testing.T) {
	r := newReveal(plain)
	r.Feed("settled block\n\nstill arriving", 40)

	full := plain("settled block", 40)
	previous := ""
	for range len(full) + 2 {
		r.Advance(1)
		got := r.View()
		if !strings.HasPrefix(full, got) {
			t.Fatalf("View = %q, not a prefix of the finished block %q", got, full)
		}
		if len(got) < len(previous) {
			t.Fatalf("View went backwards: %q then %q", previous, got)
		}
		previous = got
	}
	if previous != full {
		t.Fatalf("View = %q after playing past the end, want the whole block %q", previous, full)
	}
}

// The raw tail is never drawn: only the settled prefix reaches the screen, so
// the unsettled remainder has to be absent from the output entirely.
func TestRevealNeverDrawsTheUnsettledTail(t *testing.T) {
	r := newReveal(plain)
	r.Feed("settled block\n\nUNSETTLED TAIL", 40)
	r.Advance(1000)
	if strings.Contains(r.View(), "UNSETTLED") {
		t.Fatalf("View = %q, want the unsettled tail withheld", r.View())
	}
}

// Blocks queue behind one another and are played in arrival order. Everything
// that settles between two frames is one chunk, since it is all renderable
// together — the queue is of chunks, not of individual blocks.
func TestRevealQueuesBlocksAndPlaysThemInOrder(t *testing.T) {
	r := newReveal(plain)
	r.Feed("one\n\ntail", 40)
	r.Feed("one\n\ntwo\n\ntail", 40)
	r.Flush()

	want := plain("one", 40) + "\n\n" + plain("two", 40)
	if got := r.View(); got != want {
		t.Fatalf("View = %q, want %q", got, want)
	}
}

// Flush is what a host calls when the turn completes. An answer that has
// finished arriving must not still be typing itself out.
func TestFlushSnapsToTheEndOfWhatHasSettled(t *testing.T) {
	r := newReveal(plain)
	r.Feed("one\n\ntail", 40)
	r.Feed("one\n\ntwo\n\ntail", 40)
	r.Flush()
	if r.Pending() != 0 {
		t.Fatalf("Pending = %d after Flush, want 0", r.Pending())
	}
	if !strings.Contains(r.View(), plain("two", 40)) {
		t.Fatalf("View = %q, want everything settled shown", r.View())
	}
}

// A retried turn or a cleared buffer is a different document. Playing new
// blocks on top of the old queue would splice two answers together.
func TestRevealResetsWhenTheBufferIsRewritten(t *testing.T) {
	r := newReveal(plain)
	r.Feed("initial\n\ntail", 40)
	r.Flush()
	r.Feed("replaced\n\ntail", 40)
	r.Flush()

	got := r.View()
	if strings.Contains(got, "initial") {
		t.Fatalf("View = %q, want the previous document dropped", got)
	}
	if !strings.Contains(got, "replaced") {
		t.Fatalf("View = %q, want the new document played", got)
	}
}

// Resize re-renders every block at the new width. Blocks already played stay
// played, since that position is counted in source blocks and survives.
func TestRevealRerendersEveryBlockOnResize(t *testing.T) {
	r := newReveal(plain)
	r.Feed("one\n\ntail", 40)
	r.Feed("one\n\ntwo\n\ntail", 40)
	r.Flush()
	r.Feed("one\n\ntwo\n\ntail", 80)

	got := r.View()
	if strings.Contains(got, "<40>") {
		t.Fatalf("View = %q, want every block laid out at the new width", got)
	}
	if want := 2; strings.Count(got, "<80>") != want {
		t.Fatalf("View = %q, want %d blocks at width 80", got, want)
	}
}

// Without this the lag grows without bound: bytes arrive faster than a fixed
// clock plays them, and by the end of a long answer the display is minutes
// behind what has already been received.
func TestStepScalesWithTheBacklog(t *testing.T) {
	r := newReveal(plain)
	r.Feed(strings.Repeat("a settled paragraph\n\n", 200)+"tail", 40)

	if got := r.Step(1, 100); got <= 1 {
		t.Fatalf("Step = %d under a large backlog, want it above the base rate", got)
	}
	r.Flush()
	if got := r.Step(1, 100); got != 1 {
		t.Fatalf("Step = %d with an empty queue, want the base rate", got)
	}
}

// Cutting styled output by bytes would slice an escape sequence in half and
// spray the terminal, or drop a closing reset and bleed colour down the screen.
func TestSliceCellsCutsOnCellsAndKeepsEscapesIntact(t *testing.T) {
	const styled = "\x1b[1mbold text\x1b[0m"
	for n := range ansi.StringWidth(styled) + 1 {
		got := sliceCells(styled, n)
		if width := ansi.StringWidth(got); width != n {
			t.Fatalf("sliceCells(%d) = %q, %d cells wide", n, got, width)
		}
		if visible := ansi.Strip(got); !strings.HasPrefix("bold text", visible) {
			t.Fatalf("sliceCells(%d) shows %q, not a prefix of the text", n, visible)
		}
		if strings.Count(got, "\x1b[0m") != strings.Count(styled, "\x1b[0m") {
			t.Fatalf("sliceCells(%d) = %q, dropped the closing reset", n, got)
		}
	}
}

// A line break costs a cell, so a blank line between blocks plays out rather
// than snapping shut.
func TestSliceCellsCountsLineBreaksAsCells(t *testing.T) {
	const frame = "ab\ncd"
	if got, want := cellsIn(frame), 5; got != want {
		t.Fatalf("cellsIn = %d, want %d", got, want)
	}
	for n, want := range map[int]string{0: "", 2: "ab", 3: "ab\n", 4: "ab\nc", 5: frame} {
		if got := sliceCells(frame, n); got != want {
			t.Fatalf("sliceCells(%d) = %q, want %q", n, got, want)
		}
	}
}

// The measurement the note asks for. A block settles nothing until it closes,
// so while a long one is arriving the queue empties and the display stops —
// the freeze this mode trades reflow away for. Block size is what decides how
// bad that is, so both a chat-sized document and one full of long fences and
// tables are run, and the numbers logged for the write-up.
//
// The assertion is only that the stall is real. A change that removes it would
// invalidate the note's central objection to this design, and should be noticed
// here first.
func TestRevealStallsWhileALongBlockIsStillArriving(t *testing.T) {
	for _, block := range []int{3, 12} {
		t.Run(fmt.Sprintf("block=%d", block), func(t *testing.T) {
			stat := simulate(t, block)
			if stat.stalled == 0 {
				t.Fatal("no stall observed; the note's central objection to this mode may no longer hold")
			}
			// Backpressure has to hold the lag down, or the display finishes
			// long after the stream does.
			if stat.backlog > simCPS {
				t.Fatalf("backlog %d cells at the end, want the catch-up rule to hold it under %d", stat.backlog, simCPS)
			}
		})
	}
}

const (
	simWidth = 72
	simCPS   = 200
	simTicks = simCPS * 20 // twenty seconds of stream
)

type simStats struct {
	stalled, worst, peak, backlog int
	freezes                       int // stalls long enough to read as a freeze
}

// visible is how long the display has to sit still before a stall stops being
// "caught up for a moment" and starts reading as a hang.
const visible = 500 * time.Millisecond

// simulate runs the reveal headlessly over a seeded stream, one character in
// and one display tick out per step, which is what the demo does per frame.
func simulate(t *testing.T, block int) simStats {
	t.Helper()

	r := newReveal(markdown.New(tuikit.DefaultTheme()))
	gen := newGenerator(7, block)

	var (
		buffer, pending string
		stat            simStats
		run             int
	)
	for range simTicks {
		if pending == "" {
			pending = gen.next()
		}
		buffer, pending = buffer+pending[:1], pending[1:]

		r.Feed(buffer, simWidth)
		r.Advance(r.Step(1, simCPS)) // drain whatever is queued within a second
		stat.peak = max(stat.peak, r.Pending())

		if r.Stalled() {
			stat.stalled, run = stat.stalled+1, run+1
			stat.worst = max(stat.worst, run)
			if run == int(visible/(time.Second/simCPS)) {
				stat.freezes++
			}
		} else {
			run = 0
		}
	}
	stat.backlog = r.Pending()

	tick := time.Second / simCPS
	t.Logf("%s of stream at %d cps: stalled %s (%d%%), %d freezes over %s, longest %s, peak backlog %d cells",
		time.Duration(simTicks)*tick, simCPS,
		time.Duration(stat.stalled)*tick, stat.stalled*100/simTicks,
		stat.freezes, visible,
		time.Duration(stat.worst)*tick, stat.peak)
	return stat
}
