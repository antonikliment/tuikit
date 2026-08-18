# Revealing rendered markdown on a clock

Status: **prototyped in the demo; not recommended for the kit.** The scan fix
landed, the reveal clock was built behind `-reveal` in `examples/streaming`, and
it measures worse than the raw tail on the case that matters. What follows is
the original reasoning, then what the prototype actually showed.

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

## Measured: the scan was quadratic — fixed

`boundaryOf` re-scanned the whole prefix for link-reference definitions at every
candidate blank line, and it runs every frame even when the cache hits. It now
carries a link-reference flag and the last non-blank line through a single
forward pass, and holds a candidate cut until the following line proves it is
not a setext underline — the only rule that needed to look ahead.
`BenchmarkBoundaryOf`, on a corpus of settled blocks:

| Buffer | Before | After |
| --- | --- | --- |
| 1 KB | 24 µs | 7 µs |
| 9 KB | 809 µs | 45 µs |
| 74 KB | 49 ms | 368 µs |

Eight times the input is now eight times the time. Behaviour is unchanged,
checked differentially against the previous implementation over random documents
at every prefix length.

The second fix — skipping the already-settled region, which the monotonic-
advance test proves is safe — is no longer worth its state. 368 µs a frame on a
buffer nothing realistic reaches is not a cost worth caching against.

## Built: the reveal clock, behind `-reveal`

`examples/streaming/reveal.go`, roughly 200 lines, no library API:

```
go run ./examples/streaming -seed=7 -cps=200 -debug            # raw tail
go run ./examples/streaming -seed=7 -cps=200 -debug -reveal    # display clock
```

Settled chunks are rendered once as they settle and queued; a second clock plays
the queue out a cell at a time, cutting styled output on cell boundaries with
`ansi.Truncate` so no escape is ever halved. Progress is counted in whole
chunks plus an offset into the one playing, which is what makes resize
survivable: finished chunks stay finished and only the head's offset is clamped,
so a width change costs at most one chunk of position. `Step` raises the rate
with the backlog, `Flush` snaps to the end for turn completion (`f` in the
demo), and a rewritten buffer drops the queue rather than splicing two answers
together.

It delivers exactly what was predicted: **reflow is zero and no raw markdown is
ever visible.** Every claim in the section above about what the idea buys holds.

## What it cost: the stall is worse than predicted

`TestRevealStallsWhileALongBlockIsStillArriving` runs the seeded stream
headlessly at 200 cps for twenty seconds and counts the ticks where the queue is
empty while bytes are still arriving — the display frozen mid-answer.

| Blocks | Stalled | Freezes over 500 ms | Longest freeze |
| --- | --- | --- | --- |
| `-block=3` (chat-sized) | 18% | 2 | 1.8 s |
| `-block=12` (long fences) | 84% | 2 | 11.3 s |

The `-block=12` number is the one that settles it. Eleven seconds of a
completely static screen while an answer is actively streaming is the freeze the
glow author concluded was unfixable without glamour support, and this design
reintroduces it wholesale. The raw tail is showing text throughout that same
window.

Even the chat-sized case is not clean: 18% of the stream frozen, with a 1.8 s
pause. A stall is not visible if the display has merely caught up for a few
frames, which is why the table counts freezes over half a second separately —
those are the ones a viewer reads as a hang.

## Conclusion

Do not put the clock in the kit, and do not make it the demo's default. The
trade is real but it is the wrong way round: it removes reflow, which is
cosmetic and mostly invisible on prose, and buys a multi-second freeze on
exactly the construct — a long fence — that the raw tail was built to handle.

The prototype stays in the demo behind the flag. It is cheap to keep, it is the
evidence for this conclusion, and it is where the escapes would be tried:

- **The fence fallback would rescue it.** Falling back to the raw tail inside an
  open fence removes the `-block=12` case entirely, since fences are what
  produce the long stalls. The cost is that the visuals change mode mid-answer
  and both paths need maintaining.
- **Highlighting the partial fence directly** is the version worth wanting. The
  info string names the language, so chroma can lex a partial body without
  goldmark. That is the only route to streaming *inside* a fence, and it would
  improve the current design too, with or without a clock.

Neither is worth building for reflow alone. Revisit if the flash of raw markdown
turns out to bother people in practice, which the demo can now answer either
way.
