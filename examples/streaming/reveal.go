package main

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/antonikliment/tuikit"
)

// reveal is the prototype of the second idea in docs/notes/streaming-reveal.md:
// never show raw markdown at all. Bytes arrive into a buffer, whole blocks are
// rendered the moment they settle, and that *rendered* output is played out on
// a display clock a cell at a time. Because only final output is ever drawn,
// reflow goes to zero — no bullets appearing, no fence gaining a margin, no
// `**bold**` flashing before it turns bold.
//
// It lives in the demo, not the kit. The note calls this a UX bet and the demo
// the instrument that settles it: run the same -seed with and without -reveal
// and watch the two side by side. Nothing here is API, and the library is
// untouched.
//
// The known cost is a stall. A fence arriving over ten seconds settles nothing
// until it closes, so the queue empties and the display stops while bytes are
// still arriving — exactly the freeze the raw tail exists to avoid. That is not
// papered over here: [reveal.Stalled] reports it so it can be read off the
// status bar rather than argued about.
type reveal struct {
	render tuikit.RenderFunc

	// boundary is a StreamingMarkdown used only for its cut. It is given a
	// renderer that returns nothing, so it costs a scan and no formatting: the
	// reveal renders each block itself, once, as the block settles, rather than
	// re-rendering the whole prefix every time it grows.
	boundary tuikit.StreamingMarkdown

	width  int
	source []string // settled source blocks, in arrival order
	frame  []string // source[i] rendered at width
	taken  int      // blocks fully revealed
	cells  int      // cells revealed into frame[taken]

	consumed int // bytes of the buffer already cut into source
	settled  string
}

func newReveal(render tuikit.RenderFunc) *reveal {
	return &reveal{
		render:   render,
		boundary: tuikit.NewStreamingMarkdown(func(string, int) string { return "" }),
	}
}

// Feed hands the reveal the whole buffer as it stands. Any newly settled blocks
// are rendered at width and queued behind whatever is still playing out.
func (r *reveal) Feed(buffer string, width int) {
	if width <= 0 {
		return
	}
	if width != r.width {
		r.resize(width)
	}
	// A rewritten buffer — the demo's clear key, a retried turn — is not an
	// extension of what was revealed, so the queue is dropped rather than
	// played on top of another document's output. Same discipline as
	// StreamingMarkdown's cache, for the same reason.
	if !strings.HasPrefix(buffer, r.settled) {
		r.reset()
	}

	r.boundary.Render(buffer, width)
	cut := min(r.boundary.Settled(), len(buffer))
	if cut <= r.consumed {
		return
	}
	block := buffer[r.consumed:cut]
	r.consumed, r.settled = cut, buffer[:cut]
	r.source = append(r.source, block)
	r.frame = append(r.frame, r.renderBlock(block))
}

// renderBlock formats one settled chunk on its own. That is safe only because
// boundaryOf refuses to cut anywhere the surrounding document could still
// change the reading — inside a list, after a link-reference definition, before
// a setext underline — so a chunk renders the same alone as it would in place.
func (r *reveal) renderBlock(block string) string {
	return strings.Trim(r.render(block, r.width), "\n")
}

// Advance moves the head forward by n cells, spilling from one block into the
// next. It stops early when the queue runs dry — the display has caught up with
// what has settled, and there is nothing final left to draw.
func (r *reveal) Advance(n int) {
	for n > 0 && r.taken < len(r.frame) {
		room := cellsIn(r.frame[r.taken]) - r.cells
		if n < room {
			r.cells += n
			return
		}
		n -= room
		r.taken, r.cells = r.taken+1, 0
		if r.taken < len(r.frame) {
			// The blank line between two blocks is a cell of its own, so the
			// gap plays out rather than snapping shut.
			n--
		}
	}
}

// Flush snaps the display to the end of what has settled. A host calls this
// when the turn completes: an answer that has finished arriving should not
// still be typing itself out.
func (r *reveal) Flush() {
	r.taken, r.cells = len(r.frame), 0
}

// Pending reports how many cells are rendered but not yet shown. It is the
// backlog [reveal.Step] works against, and the number that tells you whether
// the display is running behind the stream or ahead of it.
func (r *reveal) Pending() int {
	total := 0
	for i := r.taken; i < len(r.frame); i++ {
		total += cellsIn(r.frame[i]) + 1
	}
	return max(0, total-r.cells-1)
}

// Settled reports how many bytes of the buffer have been cut into blocks and
// rendered. The rest has arrived but cannot be drawn yet.
func (r *reveal) Settled() int { return r.consumed }

// Stalled reports that everything settled has been shown. While bytes are still
// arriving that means the display is frozen mid-answer — the cost this design
// trades the raw tail for, and the thing to watch for when judging it.
func (r *reveal) Stalled() bool { return r.Pending() == 0 }

// Step is how many cells to move this tick: the base rate, raised so that
// whatever is queued clears within catchup ticks. Without that the lag grows
// without bound — bytes arrive faster than a fixed clock plays them, and by the
// end of a long answer the display is minutes behind.
func (r *reveal) Step(base, catchup int) int {
	if catchup <= 0 {
		return base
	}
	return max(base, r.Pending()/catchup)
}

// View is the revealed output: every finished block, then the part of the
// current one the head has reached.
func (r *reveal) View() string {
	shown := make([]string, 0, r.taken+1)
	shown = append(shown, r.frame[:r.taken]...)
	if r.taken < len(r.frame) && r.cells > 0 {
		shown = append(shown, sliceCells(r.frame[r.taken], r.cells))
	}
	return strings.Join(shown, "\n\n")
}

func (r *reveal) resize(width int) {
	r.width = width
	for i, block := range r.source {
		r.frame[i] = r.renderBlock(block)
	}
	// Blocks already played stay played — that position is a count of source
	// blocks, and survives a width change. Only the head's offset into the
	// block it is inside is in rendered coordinates, and re-rendering moves it,
	// so it is clamped. The visible cost is bounded by one block, which is why
	// progress is tracked per block rather than as one offset into the whole
	// document.
	if r.taken < len(r.frame) {
		r.cells = min(r.cells, cellsIn(r.frame[r.taken]))
	}
}

func (r *reveal) reset() {
	r.source, r.frame = nil, nil
	r.taken, r.cells, r.consumed, r.settled = 0, 0, 0, ""
}

// cellsIn is the length of a rendered block in reveal steps: its visible width,
// with each line break counting as one cell so vertical space plays out at the
// same rate as text.
func cellsIn(frame string) int {
	total := 0
	for i, line := range strings.Split(frame, "\n") {
		if i > 0 {
			total++
		}
		total += ansi.StringWidth(line)
	}
	return total
}

// sliceCells returns the first n cells of a rendered block. It cuts on cell
// boundaries rather than bytes, so a styled block is never cut mid-escape and
// never leaks a colour past the head — ansi.Truncate carries the sequences
// through and drops only the printable text beyond the cut.
func sliceCells(frame string, n int) string {
	lines := strings.Split(frame, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		if i > 0 {
			n--
			if n < 0 {
				break
			}
		}
		width := ansi.StringWidth(line)
		if n >= width {
			out = append(out, line)
			n -= width
			continue
		}
		out = append(out, ansi.Truncate(line, n, ""))
		break
	}
	return strings.Join(out, "\n")
}
