# Revealing rendered markdown on a clock

Status: **explored, not built.** A note so the idea and its costs survive; the
next step is a forked experiment, not a library change.

## Where this comes from

`StreamingMarkdown` renders settled blocks and shows the unfinished tail as raw
text. The tail is transient — the next frame replaces it — which is what makes
streaming safe here where it was not safe for a pipe. The residual cost is that
a settling block *reflows*: a list gains bullets and indent, a fence gains a
margin, a table gains borders and two lines of height. Raw markdown is also
briefly visible: `**bold**`, backticks, fence markers.

Two ideas have been raised for reducing that.

1. **A buffer** — hold back the cut until N bytes past the boundary.
2. **Mimicked streaming** — never show raw text at all. Buffer the arriving
   bytes, render whole blocks as they complete, and reveal that *rendered*
   output progressively on a display clock. The buffer provides a lead: bytes
   arrive faster than they are revealed, so a finished block is usually queued
   while the display is still playing out the previous one.

## On the buffer alone: no

A buffer does not remove reflow, it batches it — fewer, larger jumps, for the
same total change in layout. It is also the exact knob that sank glow's `--flow`
proposal ([PR #823](https://github.com/charmbracelet/glow/pull/823)): the
maintainer's objection was that having the user pick a buffer size "is a bit
weird", and their testing found small values corrupted output.

Most settling is invisible anyway. A prose paragraph wraps identically raw and
rendered with no indent, so nothing moves — only the inline styling arrives.
Reflow is confined to lists, fences, tables and block quotes, which argues for a
construct-specific fix rather than a global delay.

## On mimicked streaming: real, but it trades our best case away

What it genuinely buys, and no cheaper option does: **reflow goes to zero.** Not
reduced — eliminated, because only already-final output is ever drawn. The
raw-markdown flash goes with it. That is strictly better than giving the tail
the shape its block will have once settled (spec'd in the comment on `Render`),
which only approximates the final form.

The cost is the case this design currently wins on. A forty-line fence arriving
over ten seconds produces nothing renderable until it closes, so the reveal
queue empties and the display stalls — reintroducing exactly the freeze the glow
author concluded was unfixable without support from glamour. Today that fence
streams as plain text and snaps to highlighted.

Escapes, in increasing order of work:

- Fall back to the raw tail inside an open fence. Cheap, but the visuals change
  mode mid-answer — styled reveal, raw, styled — and both paths need
  maintaining.
- Highlight the partial fence directly. The info string names the language, so
  chroma can lex the partial body without goldmark, and the fence can stream
  *highlighted*. Fences are the one construct where partial rendering is
  genuinely tractable. It is a second rendering path.

## The parts that are harder than they look

**Resize.** The reveal position lives in rendered coordinates, and those do not
survive a width change — re-render at a new width and character offset N is
somewhere else entirely. The position has to be tracked in *source* coordinates
and the output re-rendered and re-sliced on every resize. Slicing styled output
also has to avoid cutting an escape sequence in half; `x/ansi` handles that
part.

**It stops being a pure function.** `Render(text, width)` is currently a
function of its arguments plus a cache. A reveal clock makes it a stateful
animation that the host must drive with ticks, which is a real API shift and sits
against the kit's norm that a component takes size as an argument and keeps no
state of its own.

**Backpressure.** If arrival outruns reveal, the lag grows without bound. The
display rate has to scale with the backlog, and snap to the end when the turn
completes. Standard, but it is policy that has to be designed and tuned.

**Background rendering is probably a non-problem.** Rendering one block through
glamour has not been measured here, but at roughly one block per few hundred
milliseconds it is not plausibly the bottleneck. The substantive idea is the
display clock; the asynchrony is not load-bearing. Measure before building it.

## Measured: the scan is quadratic

Separately from any of the above, `boundaryOf` re-scans the whole prefix for
link-reference definitions at every candidate blank line, and it runs every
frame even when the cache hits.

| Buffer | Time per call |
| --- | --- |
| 1 KB | 1.9 µs |
| 6 KB | 25 µs |
| 24 KB | 301 µs |
| 97 KB | 4.6 ms |

Four times the input, roughly thirteen times the time. A single answer is
usually 5–20 KB, so 25–300 µs a frame — unnoticeable. It only bites if the
component is pointed at a whole transcript or a tailed file.

Two fixes, both invisible to the user and both better than a buffer, which would
only mask it:

- Track whether a link-reference has been seen during the single forward scan
  instead of re-scanning the prefix at each candidate. Removes the quadratic.
- Skip the already-settled region. A cut can only move forward — the
  monotonic-advance test in `mdboundary_test.go` already proves that invariant.

## Suggested order

1. Fix the scan. Invisible, removes the only measured cost.
2. Fork and prototype the reveal clock **in the demo**, behind a flag, with the
   same `-seed` so both modes can be watched side by side. No library API
   involved. This is a UX bet, and the demo is the instrument that settles it.
3. Only if it clearly feels better: decide on the fence fallback, the resize
   handling, and whether the component grows a clock — or whether the cheaper
   tail-shaping gets close enough.
