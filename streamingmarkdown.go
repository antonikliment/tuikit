package tuikit

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// StreamingMarkdown renders markdown that is still arriving — an LLM answer, a
// tailed document — without waiting for the stream to end. Markdown parsers
// need whole blocks: an unclosed fence swallows everything after it, so text
// rendered mid-block comes out wrong. The usual workaround is to show raw text
// until the stream closes and format it all at once, which is why so many
// terminal chat UIs "pop" into formatting at the end of a turn.
//
// StreamingMarkdown splits the buffer instead. Everything up to the last
// provably-safe block boundary is rendered and cached; the unsettled remainder
// is wrapped as raw text and redrawn each frame. Because a TUI repaints
// continuously, that tail only has to be readable, never correct — it is
// replaced by real output the moment its block closes. A long code fence
// streams as plain text and snaps to highlighted when it closes, rather than
// sitting blank until then.
//
// StreamingMarkdown owns no colors and no markdown engine: it takes a
// [RenderFunc] and holds only a cache, so the root package stays free of a
// markdown dependency. See [github.com/antonikliment/tuikit/markdown] for a
// glamour-backed implementation, or supply your own.
type StreamingMarkdown struct {
	render RenderFunc

	// Set by [WithSpeculativeTail]: render the unsettled tail as markdown,
	// with its open constructs synthetically closed, instead of wrapping it
	// verbatim.
	speculate bool

	// Cache of the settled prefix: the source it was built from, the width it
	// was laid out at, and the result. Keeping the source is what makes a
	// rewritten buffer — an edited prompt, a retried turn, a replayed history
	// entry — miss instead of silently reusing another message's output.
	source   string
	width    int
	rendered string

	// The tail's own cache. It is separate because it turns over constantly —
	// every frame, against the prefix's once a block — and because a miss here
	// costs a render of a few hundred bytes rather than of the whole document.
	tailSource   string
	tailWidth    int
	tailRendered string
}

// StreamingOption configures a [StreamingMarkdown].
type StreamingOption func(*StreamingMarkdown)

// WithSpeculativeTail renders the unsettled tail as markdown instead of showing
// it raw, by appending synthetic closers for whatever constructs it leaves open
// — the trick bracket matchers use, applied to "a **bold" so the emphasis has
// something to close against. The closers are recomputed from scratch each
// frame, so they never persist into settled output.
//
// It buys two things the default gives up: no raw markdown is ever visible, and
// a partial fence arrives *highlighted* rather than as plain text, since
// CommonMark closes an open fence at the end of the document and a highlighting
// renderer will lex what it has. Reflow shrinks with it, because the tail
// already has the shape its block will settle into.
//
// The cost is a render of the tail on every frame it changes, against a wrap.
// See BenchmarkRenderTail for the measurement on this repo's glamour renderer;
// budget for it before turning this on over a fast stream.
func WithSpeculativeTail() StreamingOption {
	return func(s *StreamingMarkdown) { s.speculate = true }
}

// RenderFunc renders complete markdown at a wrap width. It is only ever called
// with whole blocks, never with a partial one.
type RenderFunc func(text string, width int) string

// NewStreamingMarkdown returns a StreamingMarkdown that formats settled blocks
// with render.
func NewStreamingMarkdown(render RenderFunc, opts ...StreamingOption) StreamingMarkdown {
	s := StreamingMarkdown{render: render}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// Render lays out the buffer at the given width: settled blocks formatted, the
// rest wrapped verbatim. Pass the whole buffer every frame — StreamingMarkdown
// keeps no copy of the text, so a caller is free to edit, truncate or replace
// it between calls, and size is an argument rather than state, matching the
// rest of the kit.
//
// The result carries no styling of its own; wrap it in a [lipgloss.Style] to
// color it.
//
// ponytail: the tail is deliberately unstyled and unindented, so a settling
// block visibly reflows — a list gains bullets and indent, a fence gains a
// margin. The alternative is to give the tail the shape its block will have
// once settled, which removes the jump but needs a second scan beside
// boundaryOf, tracking what the tail is *inside* rather than only where it is
// safe to cut. That scan now exists: see [WithSpeculativeTail], which is opt-in
// because it trades a wrap for a render on every frame the tail changes.
func (s *StreamingMarkdown) Render(text string, width int) string {
	if width <= 0 {
		return ""
	}
	cut := boundaryOf(text)
	if cut <= 0 {
		return s.tail(text, width)
	}
	if prefix := text[:cut]; s.width != width || s.source != prefix {
		s.renderPrefix(prefix, width)
	}
	tail := strings.TrimRight(text[cut:], "\n")
	if strings.TrimSpace(tail) == "" {
		return s.rendered
	}
	return s.rendered + "\n\n" + s.tail(tail, width)
}

// tail lays out the unsettled remainder: wrapped verbatim by default, or
// rendered with its open constructs closed under [WithSpeculativeTail].
func (s *StreamingMarkdown) tail(tail string, width int) string {
	if !s.speculate {
		return ansi.Wrap(tail, width, "")
	}
	if s.tailWidth != width || s.tailSource != tail {
		s.tailSource, s.tailWidth = tail, width
		s.tailRendered = strings.Trim(s.render(closeOpen(tail), width), "\n")
	}
	return s.tailRendered
}

// Settled reports how many bytes of the last rendered buffer were formatted.
// The remainder was shown raw. It exists for diagnostics — a host that wants to
// mark the boundary, or report how far behind the formatting is running.
func (s *StreamingMarkdown) Settled() int { return len(s.source) }

func (s *StreamingMarkdown) renderPrefix(prefix string, width int) {
	s.rendered = strings.TrimRight(s.render(prefix, width), "\n")
	s.source, s.width = prefix, width
}
