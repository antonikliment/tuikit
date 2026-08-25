package tuikit

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Cell is a position in a rendered frame: Row counts lines from the top, Col
// counts display columns from the left. It is the coordinate a mouse event
// arrives in, which is why nothing here knows about viewports or scroll
// offsets — the caller hands over the frame it just drew.
type Cell struct{ Row, Col int }

// Selection is a drag-selected range over a rendered frame. It holds two cells
// and nothing else: the frame is passed to [Selection.Paint] and
// [Selection.Text] each time, so a selection never goes stale against content
// that has been re-rendered underneath it.
//
// Selection is linear by default, the way a text editor and a terminal are:
// the first row runs from the anchor to the end of the line, whole rows
// follow, and the last row ends at the cursor. That is right for a transcript
// and wrong for a frame laid out in columns, where a drag inside one panel
// still picks up the panels beside it. Set [Selection.Block] for that case: it
// clamps every row to the dragged column range, so the selection is the
// rectangle the mouse drew. Hosts usually bind it to alt-drag, the terminal
// convention.
//
// The zero value is an empty linear selection, ready to use.
type Selection struct {
	// Block selects a rectangle rather than a run of lines. It can be flipped
	// mid-drag: nothing is cached, so the next Paint or Text just reads it.
	Block bool

	anchor Cell
	cursor Cell
	active bool
}

// Begin anchors a new selection at c, as a mouse press does.
func (s *Selection) Begin(c Cell) {
	s.anchor, s.cursor, s.active = c, c, true
}

// Extend moves the loose end to c, as a drag does. It is a no-op until Begin.
func (s *Selection) Extend(c Cell) {
	if s.active {
		s.cursor = c
	}
}

// Clear drops the selection, keeping the Block mode the host set.
func (s *Selection) Clear() { *s = Selection{Block: s.Block} }

// Empty reports whether there is nothing to paint or copy — no selection, or
// one that never moved off its anchor (a plain click).
func (s Selection) Empty() bool { return !s.active || s.anchor == s.cursor }

// Paint returns view with the selected range drawn in style. Rows and columns
// outside the frame are ignored, so a drag past the last line is harmless.
func (s Selection) Paint(view string, style lipgloss.Style) string {
	if s.Empty() {
		return view
	}
	lines := strings.Split(view, "\n")
	for row, line := range lines {
		left, right, ok := s.span(row, ansi.StringWidth(line))
		if !ok {
			continue
		}
		// ponytail: the highlight is plain text on style's background — the
		// selected slice keeps its width but loses its own colors until the
		// drag ends. Re-emitting the SGR runs inside the slice is the upgrade
		// path, and is only worth it if the color loss actually bothers anyone.
		lines[row] = ansi.Cut(line, 0, left) +
			ansi.ResetStyle + style.Render(ansi.Strip(ansi.Cut(line, left, right))) +
			ansi.Cut(line, right, ansi.StringWidth(line))
	}
	return strings.Join(lines, "\n")
}

// Text is the plain text of the selected range, newline-separated and stripped
// of escapes — what a copy gesture puts on the clipboard. Trailing blanks are
// trimmed per line, because a selection that runs past the end of a line is
// selecting the padding, not the text.
func (s Selection) Text(view string) string {
	if s.Empty() {
		return ""
	}
	var out []string
	for row, line := range strings.Split(view, "\n") {
		left, right, ok := s.span(row, ansi.StringWidth(line))
		if !ok {
			continue
		}
		out = append(out, strings.TrimRight(ansi.Strip(ansi.Cut(line, left, right)), " "))
	}
	return strings.Join(out, "\n")
}

// span is the selected column range on row, in a line that is width columns
// wide. It reports false when the row is outside the selection or the range
// collapses to nothing.
func (s Selection) span(row, width int) (int, int, bool) {
	start, end := s.bounds()
	if row < start.Row || row > end.Row {
		return 0, 0, false
	}
	left, right := 0, width
	switch {
	case s.Block:
		left, right = min(s.anchor.Col, s.cursor.Col), max(s.anchor.Col, s.cursor.Col)
	default:
		if row == start.Row {
			left = start.Col
		}
		if row == end.Row {
			right = end.Col
		}
	}
	left, right = clamp(left, width), clamp(right, width)
	return left, right, left < right
}

// bounds returns the selection in reading order, so dragging up or leftwards
// paints the same range as dragging the other way.
func (s Selection) bounds() (Cell, Cell) {
	a, b := s.anchor, s.cursor
	if b.Row < a.Row || (b.Row == a.Row && b.Col < a.Col) {
		a, b = b, a
	}
	return a, b
}

func clamp(col, width int) int {
	if col < 0 {
		return 0
	}
	if col > width {
		return width
	}
	return col
}
