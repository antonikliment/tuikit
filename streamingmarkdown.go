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

	// Set by [WithRawTail]: wrap the unsettled tail verbatim instead of
	// rendering it with its open constructs closed.
	rawTail bool

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

// WithRawTail wraps the unsettled tail verbatim instead of rendering it, which
// is what the component did before speculative closing existed.
//
// The default costs a render of the tail on every frame it changes, where this
// costs a wrap: 129µs against 3.2µs in BenchmarkRenderTail. That is immaterial
// against a 16ms frame in one pane, and worth declining in a program repainting
// several of them at speed. The visible difference is markers — "**bold" and
// backticks appear on screen — and a partial fence arriving as plain text
// rather than highlighted.
func WithRawTail() StreamingOption {
	return func(s *StreamingMarkdown) { s.rawTail = true }
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

// Render lays out the buffer at the given width: settled blocks formatted, and
// the unsettled tail rendered with its open constructs closed (or wrapped
// verbatim under [WithRawTail]). Pass the whole buffer every frame — StreamingMarkdown
// keeps no copy of the text, so a caller is free to edit, truncate or replace
// it between calls, and size is an argument rather than state, matching the
// rest of the kit.
//
// The result carries no styling of its own; wrap it in a [lipgloss.Style] to
// color it.
//
// Block-level reflow survives either way: the tail is still a partial block, so
// a list gains its bullet and a table its borders when the block settles. What
// speculative closing removes is the inline half of that jump, and the markers.
func (s *StreamingMarkdown) Render(text string, width int) string {
	if width <= 0 {
		return ""
	}
	cut := boundaryOf(text)
	if cut <= 0 {
		return s.tail(text, width)
	}
	if prefix := text[:cut]; s.width != width || s.source != prefix {
		s.rendered = strings.TrimRight(s.render(prefix, width), "\n")
		s.source, s.width = prefix, width
	}
	tail := strings.TrimRight(text[cut:], "\n")
	if strings.TrimSpace(tail) == "" {
		return s.rendered
	}
	return s.rendered + "\n\n" + s.tail(tail, width)
}

// tail lays out the unsettled remainder: rendered with its open constructs
// closed, or wrapped verbatim under [WithRawTail].
func (s *StreamingMarkdown) tail(tail string, width int) string {
	if s.rawTail {
		return ansi.Wrap(tail, width, "")
	}
	if s.tailWidth != width || s.tailSource != tail {
		s.tailSource, s.tailWidth = tail, width
		s.tailRendered = strings.Trim(s.render(closeOpen(tail), width), "\n")
	}
	return s.tailRendered
}

// Settled reports how many bytes of the last rendered buffer had provably
// settled, and so were rendered from their own source rather than speculatively
// closed. It exists for diagnostics — a host that wants to mark the boundary,
// or report how far behind the formatting is running.
func (s *StreamingMarkdown) Settled() int { return len(s.source) }
