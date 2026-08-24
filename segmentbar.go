package tuikit

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// Segment is one attributed slice of a [SegmentBar]: a legend label, a
// preformatted value cell, its share of the whole in [0, 1], and the style
// painting both its bar cells and its legend swatch.
type Segment struct {
	Label string
	Value string
	Share float64
	Style lipgloss.Style
}

// SegmentBar renders a width-cell horizontal bar split proportionally by the
// segments' shares, with a swatch legend of label/value/percent rows beneath.
// Every segment with a positive share paints at least one cell so tiny slices
// stay visible; free absorbs whatever the segments leave of the bar and closes
// the legend as the "□" row.
func SegmentBar(width int, segments []Segment, free Segment) string {
	var bar strings.Builder
	used := 0
	rows := make([][]string, 0, len(segments)+1)
	for _, segment := range segments {
		cells := int(segment.Share*float64(width) + 0.5)
		if segment.Share > 0 {
			cells = max(cells, 1)
		}
		cells = min(cells, width-used)
		used += cells
		bar.WriteString(segment.Style.Render(strings.Repeat("█", cells)))
		rows = append(rows, legendRow("■", segment))
	}
	bar.WriteString(free.Style.Render(strings.Repeat("░", width-used)))
	rows = append(rows, legendRow("□", free))

	widths := Columns(rows)
	var out strings.Builder
	out.WriteString(bar.String() + "\n")
	for _, row := range rows {
		out.WriteString(JoinCells(row, widths, 2) + "\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func legendRow(swatch string, segment Segment) []string {
	return []string{segment.Style.Render(swatch + " " + segment.Label), segment.Value, fmt.Sprintf("%.1f%%", segment.Share*100)}
}
