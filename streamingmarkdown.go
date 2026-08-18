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

	// Cache of the settled prefix: the source it was built from, the width it
	// was laid out at, and the result. Keeping the source is what makes a
	// rewritten buffer — an edited prompt, a retried turn, a replayed history
	// entry — miss instead of silently reusing another message's output.
	source   string
	width    int
	rendered string
}

// RenderFunc renders complete markdown at a wrap width. It is only ever called
// with whole blocks, never with a partial one.
type RenderFunc func(text string, width int) string

// NewStreamingMarkdown returns a StreamingMarkdown that formats settled blocks
// with render.
func NewStreamingMarkdown(render RenderFunc) StreamingMarkdown {
	return StreamingMarkdown{render: render}
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
// once settled, which removes the jump but needs partial-construct
// classification: a second parser beside boundaryOf, tracking what the tail is
// *inside* rather than only where it is safe to cut. Worth revisiting if the
// reflow reads badly in practice; not worth the parser before then.
func (s *StreamingMarkdown) Render(text string, width int) string {
	if width <= 0 {
		return ""
	}
	cut := boundaryOf(text)
	if cut <= 0 {
		return ansi.Wrap(text, width, "")
	}
	if prefix := text[:cut]; s.width != width || s.source != prefix {
		s.renderPrefix(prefix, width)
	}
	tail := strings.TrimRight(text[cut:], "\n")
	if strings.TrimSpace(tail) == "" {
		return s.rendered
	}
	return s.rendered + "\n\n" + ansi.Wrap(tail, width, "")
}

// Settled reports how many bytes of the last rendered buffer were formatted.
// The remainder was shown raw. It exists for diagnostics — a host that wants to
// mark the boundary, or report how far behind the formatting is running.
func (s *StreamingMarkdown) Settled() int { return len(s.source) }

func (s *StreamingMarkdown) renderPrefix(prefix string, width int) {
	s.rendered = strings.TrimRight(s.render(prefix, width), "\n")
	s.source, s.width = prefix, width
}
