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

// The measurement. A block settles nothing until it closes, so while a long one
// is arriving there is nothing final to draw — the freeze this design trades
// reflow away for, and what the reserve exists to cover.
//
// A freeze is measured as a run of ticks in which the visible head does not
// move. That is deliberately not the same as a tick that advanced no cells:
// while the policy is conserving it moves the head by well under a cell a tick,
// and counting those as frozen would call a working display broken. Nor is it
// the same as an empty queue, which is reported separately as an underrun.
func TestRevealPacing(t *testing.T) {
	for _, block := range []int{3, 12} {
		t.Run(fmt.Sprintf("block=%d", block), func(t *testing.T) {
			eager := simulate(t, block, policy{base: 1, catchup: simCPS, hold: 0})
			held := simulate(t, block, held())

			if eager.worst == 0 {
				t.Fatal("no freeze without a reserve; the objection this series answers may no longer hold")
			}
			if held.worst >= eager.worst {
				t.Fatalf("longest freeze %d ticks with a reserve and %d without, want the reserve to shorten it",
					held.worst, eager.worst)
			}
			// The reserve is deliberate latency; the catch-up rule governs the
			// rest, or the display finishes long after the stream does.
			if limit := held.reserve + simCPS; held.backlog > limit {
				t.Fatalf("backlog %d cells at the end, want it under %d (reserve %d)", held.backlog, limit, held.reserve)
			}
		})
	}
}

// How the reserve multiplier trades freezing against lag. Logged rather than
// asserted: it is the tuning evidence for the write-up, and pinning a preferred
// value here would make the table a tautology.
func TestRevealPacingSweepsTheReserve(t *testing.T) {
	if testing.Short() {
		t.Skip("twelve twenty-second simulations")
	}
	for _, block := range []int{3, 12} {
		for _, hold := range []float64{0, 0.5, 1, 2, 4, 8} {
			stat := simulate(t, block, policy{base: 1, catchup: simCPS, hold: hold})
			tick := time.Second / simCPS
			t.Logf("block=%-2d hold=%-3.1f  longest freeze %-7s  freezes>%s %-2d  underrun %-7s  lag at end %-4dc  holding %dc",
				block, hold,
				(time.Duration(stat.worst) * tick).Round(time.Millisecond),
				visible, stat.freezes,
				(time.Duration(stat.stalled) * tick).Round(time.Millisecond),
				stat.backlog, stat.reserve)
		}
	}
}

const (
	simWidth = 72
	simCPS   = 200
	simTicks = simCPS * 20 // twenty seconds of stream
)

// visible is how long the display has to sit still before a pause stops reading
// as pacing and starts reading as a hang.
const visible = 500 * time.Millisecond

type simStats struct {
	stalled int // ticks with an empty queue: an underrun
	worst   int // longest run of ticks in which the visible head did not move
	freezes int // runs of that kind longer than visible
	backlog int // cells still unplayed at the end
	reserve int // what the policy was holding back at the end
}

// simulate runs the reveal headlessly over a seeded stream, one character in
// and one display tick out per step, which is what the demo does per frame.
func simulate(t *testing.T, block int, p policy) simStats {
	t.Helper()

	r := newReveal(markdown.New(tuikit.DefaultTheme()))
	gen := newGenerator(7, block)

	var (
		buffer, pending string
		stat            simStats
		still           int
		last            [2]int
	)
	for range simTicks {
		if pending == "" {
			pending = gen.next()
		}
		buffer, pending = buffer+pending[:1], pending[1:]

		r.Feed(buffer, simWidth)
		r.Tick(p)

		if r.Stalled() {
			stat.stalled++
		}
		taken, cells := r.head()
		if now := [2]int{taken, cells}; now == last {
			still++
			stat.worst = max(stat.worst, still)
			if still == int(visible/(time.Second/simCPS)) {
				stat.freezes++
			}
		} else {
			last, still = now, 0
		}
	}
	stat.backlog, stat.reserve = r.Pending(), r.Reserve(p)
	return stat
}
