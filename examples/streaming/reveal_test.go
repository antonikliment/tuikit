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

// A policy for tests: one cell a tick, a one-second catch-up at the simulated
// rate, and the reserve held at its estimated size.
func held() policy { return policy{base: 1, catchup: simCPS, hold: 1} }

// queue puts n cells of settled output in front of the head, with a settle
// history that makes the estimated reserve predictable.
func queue(t *testing.T, r *reveal, cells, gap int) {
	t.Helper()
	r.frame = append(r.frame, strings.Repeat("x", cells))
	r.source = append(r.source, "x")
	for range 3 {
		for range gap {
			r.gaps.Tick()
		}
		r.gaps.Settled()
	}
}

// Below the reserve the drain scales with what is left, so the queue decays
// toward empty rather than hitting it. This is the whole fix: a display that
// keeps moving through a drought instead of freezing.
func TestRateSlowsDownAsTheQueueEmpties(t *testing.T) {
	r := newReveal(plain)
	queue(t, r, 400, 200)

	full := r.rate(held())
	r.Advance(350)
	nearlyEmpty := r.rate(held())

	if nearlyEmpty >= full {
		t.Fatalf("rate = %.3f with 50 cells left and %.3f with 400, want it to slow as the queue drains", nearlyEmpty, full)
	}
	if nearlyEmpty <= 0 {
		t.Fatalf("rate = %.3f with cells still queued, want the display to keep moving", nearlyEmpty)
	}
}

// The property that replaces the stall: with nothing new arriving, the queue
// has to outlast a drought many times longer than it would have at full rate.
func TestConservingOutlastsADroughtThatWouldHaveStalled(t *testing.T) {
	conserving, eager := newReveal(plain), newReveal(plain)
	for _, r := range []*reveal{conserving, eager} {
		queue(t, r, 400, 200)
	}

	survived := func(r *reveal, p policy) int {
		for tick := range 2000 {
			if r.Stalled() {
				return tick
			}
			r.Tick(p)
		}
		return 2000
	}
	no := held()
	no.hold = 0

	long, short := survived(conserving, held()), survived(eager, no)
	if long <= short*2 {
		t.Fatalf("conserving lasted %d ticks and eager %d, want the reserve to stretch much further", long, short)
	}
	if short >= 500 {
		t.Fatalf("eager lasted %d ticks, want it to run the 400-cell queue dry quickly", short)
	}
}

// hold=0 turns the reserve off entirely, which is how the before/after
// comparison is taken in one binary.
func TestHoldZeroRestoresTheEagerBehaviour(t *testing.T) {
	r := newReveal(plain)
	queue(t, r, 400, 200)

	p := held()
	p.hold = 0
	if got := r.rate(p); got < p.base {
		t.Fatalf("rate = %.3f with the reserve off, want at least the base rate %.3f", got, p.base)
	}
}

// Above the reserve the backlog is drained rather than held, or the lag grows
// without bound: bytes arrive faster than a fixed clock plays them.
func TestRateCatchesUpOnALargeBacklog(t *testing.T) {
	r := newReveal(plain)
	queue(t, r, 100000, 10)

	if got := r.rate(held()); got <= 1 {
		t.Fatalf("rate = %.3f under a large backlog, want it above the base rate", got)
	}
}

// The open construct does not decide *when* to buffer — the reserve is held
// continuously — it decides how hard to brake. A fence has no predictable end,
// so the same queue has to be spent more slowly inside one.
func TestAFenceBrakesHarderThanProse(t *testing.T) {
	r := newReveal(plain)
	queue(t, r, 400, 200)

	r.open = openState{kind: openParagraph}
	prose := r.rate(held())
	r.open = openState{kind: openFence, open: 300}
	fence := r.rate(held())

	if fence >= prose {
		t.Fatalf("rate = %.3f inside a fence and %.3f in prose, want the fence to brake harder", fence, prose)
	}
}

// Fractions have to carry: the policy runs well below one cell a tick while
// conserving, and truncating each tick would round the rate to zero and stall
// anyway.
func TestTickCarriesFractionsOfACell(t *testing.T) {
	r := newReveal(plain)
	queue(t, r, 400, 4000) // a huge reserve, so the rate is far below one cell

	before := r.Pending()
	for range 200 {
		r.Tick(held())
	}
	if r.Pending() >= before {
		t.Fatalf("pending %d then %d after 200 ticks, want the display to have moved", before, r.Pending())
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
			stat := simulate(t, block, held())
			if stat.stalled == 0 {
				t.Fatal("no stall observed; the note's central objection to this mode may no longer hold")
			}
			// Backpressure has to hold the lag down, or the display finishes
			// long after the stream does. The reserve is deliberate latency and
			// is allowed on top of it; the catch-up rule governs the rest.
			if limit := stat.reserve + simCPS; stat.backlog > limit {
				t.Fatalf("backlog %d cells at the end, want the catch-up rule to hold it under %d (reserve %d)",
					stat.backlog, limit, stat.reserve)
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
	reserve                       int // what the policy was holding back at the end
}

// visible is how long the display has to sit still before a stall stops being
// "caught up for a moment" and starts reading as a hang.
const visible = 500 * time.Millisecond

// simulate runs the reveal headlessly over a seeded stream, one character in
// and one display tick out per step, which is what the demo does per frame.
func simulate(t *testing.T, block int, p policy) simStats {
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
		r.Tick(p)
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
	stat.backlog, stat.reserve = r.Pending(), r.Reserve(p)

	tick := time.Second / simCPS
	t.Logf("%s of stream at %d cps: stalled %s (%d%%), %d freezes over %s, longest %s, peak backlog %d cells, holding %d",
		time.Duration(simTicks)*tick, simCPS,
		time.Duration(stat.stalled)*tick, stat.stalled*100/simTicks,
		stat.freezes, visible,
		time.Duration(stat.worst)*tick, stat.peak, stat.reserve)
	return stat
}

// A word appearing at once reads as typing; a word assembling letter by letter
// reads as a machine. The head therefore stops at word ends, not wherever the
// cell budget happens to land.
func TestSnapToWordStopsAtWholeWords(t *testing.T) {
	const frame = "hello world again"
	for n, want := range map[int]int{
		0:  0,
		3:  0,  // mid-"hello": nothing whole yet
		6:  6,  // "hello " is whole
		9:  6,  // mid-"world": back to the last whole word
		12: 12, // "hello world " is whole
		17: 17, // the whole line
		40: 17, // past the end
	} {
		if got := snapToWord(frame, n); got != want {
			t.Errorf("snapToWord(%d) = %d, want %d", n, got, want)
		}
	}
}

// A line break ends a word, so a block does not hold back a finished line
// waiting for the first word of the next one.
func TestSnapToWordStopsAtLineBreaks(t *testing.T) {
	const frame = "first\nsecond"
	if got, want := snapToWord(frame, 6), 6; got != want {
		t.Fatalf("snapToWord(6) = %d, want %d — the break after \"first\"", got, want)
	}
}

// A URL or a long code token is wider than the budget will ever be at first.
// Withholding it would freeze the display until all of it arrived, which is the
// failure this whole design exists to avoid.
func TestSnapToWordShowsAWordWiderThanTheBudget(t *testing.T) {
	const frame = "https://example.com/a/very/long/path/that/never/breaks"
	if got, want := snapToWord(frame, 30), 30; got != want {
		t.Fatalf("snapToWord(30) = %d, want %d — a partial long word beats nothing", got, want)
	}
	// Up to longWord cells the head still waits, so an ordinary word is never
	// shown half-drawn.
	if got, want := snapToWord(frame, 12), 0; got != want {
		t.Fatalf("snapToWord(12) = %d, want %d — still within the wait a word is allowed", got, want)
	}
}

// Snapping counts visible cells, so styling must not shift where a word ends.
func TestSnapToWordIgnoresEscapeSequences(t *testing.T) {
	plainFrame := "hello world"
	styled := "\x1b[1mhello\x1b[0m world"
	if got, want := snapToWord(styled, 9), snapToWord(plainFrame, 9); got != want {
		t.Fatalf("snapToWord on styled text = %d, on plain = %d, want them equal", got, want)
	}
}

// The two have to compose: what the head reveals is the slice at the snapped
// offset, and it must still be a prefix of the finished block.
func TestViewRevealsWholeWordsOfTheBlockInFlight(t *testing.T) {
	r := newReveal(plain)
	r.Feed("one two three four\n\ntail", 40)

	full := plain("one two three four", 40)
	for range cellsIn(full) + 2 {
		r.Advance(1)
		got := r.View()
		if !strings.HasPrefix(full, got) {
			t.Fatalf("View = %q, not a prefix of %q", got, full)
		}
		if trailing := strings.TrimSuffix(got, " "); got != full && trailing != "" &&
			!strings.HasSuffix(got, " ") && !strings.HasPrefix(full[len(got):], " ") &&
			len(got) > len(plain("", 40)) {
			t.Fatalf("View = %q ends mid-word", got)
		}
	}
}
