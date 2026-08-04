package tuikit

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Columns measures each column of rows by its widest cell, returning one width
// per column. Short rows are allowed: the result is as long as the longest row,
// and a column no row reaches simply does not appear.
//
// Width is measured with [ansi.StringWidth] rather than len, because a styled
// cell carries escape sequences that occupy no screen columns and East Asian
// characters occupy two. Measuring with len pads by the length of the escape
// sequence and skews the whole table right — the bug every hand-rolled terminal
// table hits once.
//
// Pass the header row along with the body so a column never renders narrower
// than its own title:
//
//	widths := tuikit.Columns(append([][]string{header}, rows...))
func Columns(rows [][]string) []int {
	var widths []int
	for _, row := range rows {
		for i, cell := range row {
			for i >= len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], ansi.StringWidth(cell))
		}
	}
	return widths
}

// JoinCells lays one row out against widths, separating columns by gap spaces.
// A negative gap is treated as zero.
//
// The final cell is not padded and trailing whitespace is trimmed, so no line
// carries invisible padding into a terminal's selection or a diff. Cells beyond
// the end of widths are joined at their natural size, so a row longer than the
// measured table still renders every cell rather than dropping it.
func JoinCells(row []string, widths []int, gap int) string {
	gap = max(gap, 0)
	var b strings.Builder
	for i, cell := range row {
		if i == len(row)-1 {
			b.WriteString(cell)
			break
		}
		width := 0
		if i < len(widths) {
			width = widths[i]
		}
		b.WriteString(cell)
		b.WriteString(strings.Repeat(" ", max(0, width-ansi.StringWidth(cell))+gap))
	}
	return strings.TrimRight(b.String(), " ")
}

// Pad right-pads text with spaces to width, measuring as [Columns] does. Text
// already at or beyond width is returned untouched — Pad never truncates, so a
// value is never silently cut; reach for [TruncMiddle] first when a hard cap is
// what you want.
func Pad(text string, width int) string {
	return text + strings.Repeat(" ", max(0, width-ansi.StringWidth(text)))
}

// Widest returns the display width of the widest value, or 0 for an empty
// slice. It is the one-dimensional [Columns]: the width to [Pad] a label column
// to when the rows are key/value pairs rather than a table.
func Widest(values []string) int {
	width := 0
	for _, value := range values {
		width = max(width, ansi.StringWidth(value))
	}
	return width
}

// IndentWidth is how many spaces [Indent] adds per level. Wide enough to see,
// narrow enough that a nested block still fits an 80-column terminal.
const IndentWidth = 2

// Indent prefixes text with depth levels of [IndentWidth] spaces. A depth of
// zero or less returns text unchanged.
func Indent(text string, depth int) string {
	return strings.Repeat(" ", max(0, depth)*IndentWidth) + text
}

// IndentLines applies [Indent] to every line of text and returns the result
// with a trailing newline. Trailing blank lines in the input are dropped first,
// so indenting an already-newline-terminated block does not accumulate them.
// Empty input yields an empty string rather than a lone newline.
func IndentLines(text string, depth int) string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		b.WriteString(Indent(line, depth))
		b.WriteString("\n")
	}
	return b.String()
}
