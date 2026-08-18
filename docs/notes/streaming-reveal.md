# What to do with the unsettled tail

Status: **settled.** `StreamingMarkdown` renders the tail with its open
constructs closed. Two other designs were built and measured first; both lost,
and both survive in `examples/streaming` as the evidence.

## The problem

`StreamingMarkdown` renders blocks that have provably settled and has to do
*something* with the remainder, which is mid-construct by definition. Showing it
raw means `**bold**` and backticks on screen, and a settling block reflows: a
list gains bullets and indent, a fence a margin, a table borders and two lines of
height. Waiting instead means the answer pops into shape at the end, which is
what most terminal chat UIs do.

Most settling is invisible either way — prose wraps identically raw and rendered
— so the whole argument is about lists, fences, tables and quotes.

## What shipped: close the tail and render it

Keep a stack of the inline constructs the tail leaves open, append the closers in
reverse, render that. Bracket matching, applied to markdown. `mdspeculate.go`;
`WithRawTail()` opts out. The closers are recomputed each frame and never touch
the settled prefix, so a wrong guess is corrected within a frame and cannot
corrupt what has already been shown — which is what licenses the scan to
approximate CommonMark's inline rules rather than implement them.

**Fences are deliberately not on the stack.** CommonMark closes an open fence at
the end of the document, so a partial fence is already a code block to goldmark
and chroma lexes what has arrived. Rendering the tail is therefore enough to make
a fence stream *highlighted*; a synthetic ` ``` ` would be a no-op. The one
construct the clock below cannot stream is the one this gets free.

Cost, from `BenchmarkRenderTail`: 129 µs a frame against 3.2 µs for a wrap, of
which the scan is 1.5 µs — the rest is glamour rendering the tail. Immaterial
against a 16 ms frame in one pane; `WithRawTail` exists for a program repainting
several at speed.

It does not remove block-level reflow. The tail has its final *inline* shape, but
a list item still gains its bullet when the block settles.

## Rejected: a plain buffer

Hold the cut back N bytes past the boundary. It does not remove reflow, it
batches it — fewer, larger jumps for the same total layout change. It also puts
the size in the user's hands, which is a knob nobody can set well: too small
corrupts output, too large is indistinguishable from waiting for the turn to
end.

## Rejected: a display clock

Never draw anything unfinished: buffer arriving bytes, render whole blocks as
they settle, and reveal that *rendered* output progressively on a second clock.
`-reveal -hold=0`.

It delivers what it promises — reflow goes to **zero**, not merely reduced,
because only final output is ever drawn.

It also freezes. Nothing settles inside a fence, so the queue empties and the
display stops. From `TestRevealPacing`, 20 s of seeded stream at 200 cps:

| Blocks | Longest freeze | Freezes > 500 ms |
| --- | --- | --- |
| 3 (chat-sized) | 1.83 s | 5 |
| 12 (long fences) | **11.74 s** | 2 |

Eleven seconds of static screen mid-answer is worse than anything the raw tail
does, and it is not fixable from outside the markdown engine: the freeze lasts
exactly as long as the block takes to close. Two structural costs come with it: `Render(text, width)` stops being a pure function and becomes
an animation the host drives with ticks, and the reveal position lives in
rendered coordinates, which do not survive a resize.

## Rejected: an adaptive playout buffer on the clock

The freeze is a buffer underrun — a bursty producer feeding a constant-rate
consumer — so this borrows from the literatures that own that shape. Four pieces:
`classify` for what the tail is inside (the distinction that matters is the
fence, whose end nothing bounds); `gapEstimator`, RFC 3550's EWMA mean and
deviation over settle gaps, with `Reserve() = mean + 4·deviation` because the
gaps are badly skewed; a two-region rate map after Huang et al. (SIGCOMM 2014),
steering off buffer occupancy alone and draining *proportionally* below the
reserve so the queue decays toward empty rather than hitting it; and
word-boundary release, as Vercel's AI SDK does in `smoothStream`, because a word
assembling letter by letter reads as a machine.

The open construct sets how hard to brake — fence 3×, container 1.5×, prose 1× —
not when to buffer. A reserve grown in reaction to seeing a fence would have to
be built from bytes arriving *during* the fence, and nothing settles during a
fence. A reserve can only be spent if it was already held.

| Blocks | `-hold` | Longest freeze | Freezes > 500 ms | Underrun |
| --- | --- | --- | --- | --- |
| 3 | 0 (off) | 1.83 s | 5 | 5.55 s |
| 3 | **1** | **1.00 s** | **1** | 1.10 s |
| 3 | 8 | 1.06 s | 2 | 0.11 s |
| 12 | 0 (off) | 11.74 s | 2 | 17.29 s |
| 12 | **1** | **5.14 s** | **2** | 5.25 s |

Three readings. It **works on the realistic case** — freezes go from five to one
for about a second of steady-state lag. The response is **U-shaped**: past
`-hold=2` freezes get worse while underrun keeps falling, because the policy is
braking so hard the head crawls. And it **cannot reach the pathological case**:
at `-block=12` underrun tracks the freeze almost exactly, so the queue is
genuinely empty. There is no reserve to hold because nothing settled to put in
one. A buffer cannot manufacture material.

That is what decided it. The remaining freeze is on long fences, and no amount of
buffering reaches it — while closing the tail streams the fence highlighted
throughout that same window.

## Also fixed along the way

`boundaryOf` re-scanned the whole prefix for link-reference definitions at every
candidate blank line. It now carries a link-reference flag and the last non-blank
line through one forward pass, holding a candidate cut until the following line
proves it is not a setext underline — the only rule that needed lookahead.

| Buffer | Before | After |
| --- | --- | --- |
| 1 KB | 24 µs | 7 µs |
| 9 KB | 809 µs | 45 µs |
| 74 KB | 49 ms | 368 µs |

Quadratic to linear, behaviour unchanged, checked differentially against the
previous implementation over random documents at every prefix length.

## Reproducing

```
go run ./examples/streaming -seed=7 -cps=200 -block=12                  # what ships
go run ./examples/streaming -seed=7 -cps=200 -block=12 -raw             # the raw tail
go run ./examples/streaming -seed=7 -cps=200 -block=12 -reveal -hold=0  # the clock
go run ./examples/streaming -seed=7 -cps=200 -block=12 -reveal          # the buffer
```

The clock and its buffer stay in the demo. They are cheap to keep, they are the
evidence, and `-hold=0` reproduces the first round's numbers in the same binary.
