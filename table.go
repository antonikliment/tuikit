package tuikit

import "strings"

// Table renders header and rows as aligned columns, one line per row, with the
// header upper-cased and muted.
//
// It is the rung above [Columns] and [JoinCells]: those measure and lay out,
// this is the loop every caller writes on top of them. Widths are measured over
// the header as well as the body, so a column never renders narrower than its
// own title. Short rows are padded out by JoinCells rather than dropped.
//
// The result is newline-terminated, ready for [IndentLines] if it belongs
// inside a block.
func (t Theme) Table(paint Painter, header []string, rows [][]string, gap int) string {
	if len(rows) == 0 {
		return ""
	}
	titles := make([]string, len(header))
	for i, cell := range header {
		titles[i] = strings.ToUpper(cell)
	}
	widths := Columns(append([][]string{titles}, rows...))
	var b strings.Builder
	b.WriteString(paint(t.MutedStyle(), JoinCells(titles, widths, gap)))
	b.WriteString("\n")
	for _, row := range rows {
		b.WriteString(JoinCells(row, widths, gap))
		b.WriteString("\n")
	}
	return b.String()
}

// Pairs renders keys and values as a two-column block with the keys muted and
// padded to a common width. Order is the caller's: sort the keys first if the
// output has to be stable.
//
// Keys without a matching value render with an empty value rather than
// panicking, so a mismatched pair of slices degrades instead of crashing.
//
// The result is newline-terminated, ready for [IndentLines] if it belongs
// inside a block.
func (t Theme) Pairs(paint Painter, keys, values []string, gap int) string {
	if len(keys) == 0 {
		return ""
	}
	width := Widest(keys)
	var b strings.Builder
	for i, key := range keys {
		value := ""
		if i < len(values) {
			value = values[i]
		}
		b.WriteString(JoinCells([]string{paint(t.MutedStyle(), Pad(key, width)), value}, []int{width}, gap))
		b.WriteString("\n")
	}
	return b.String()
}
