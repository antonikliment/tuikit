package tuikit

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// StatusTitle renders a "Title ............ ● status" header line, the title in
// titleColor and the status dot in statusColor, filling width.
func (t Theme) StatusTitle(title, status string, titleColor, statusColor color.Color, width int) string {
	left := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render(title)
	right := lipgloss.NewStyle().Foreground(statusColor).Render("● " + status)
	space := max(1, max(0, width-8)-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", space) + right
}

// Rule renders a horizontal muted divider.
func (t Theme) Rule(width int) string {
	return t.MutedStyle().Render(strings.Repeat("─", max(0, width-6)))
}

// Field renders an aligned "label: value" pair.
func Field(label, value string) string { return fmt.Sprintf("%-9s %s", label+":", value) }

// VerticalSlice hard-clips content to height lines starting at offset, so a
// block never grows past a fixed footprint.
func VerticalSlice(content string, offset, height int) string {
	lines := strings.Split(content, "\n")
	if height <= 0 || len(lines) <= height {
		return content
	}
	offset = min(max(0, offset), len(lines)-height)
	return strings.Join(lines[offset:offset+height], "\n")
}

// AdaptiveWidth computes a responsive column width: it fits as many columns of
// at least min width (separated by gap) as total allows, then divides the space
// evenly, clamping each column to [min, max].
func AdaptiveWidth(total, gap, minimum, maximum int) int {
	columns := max(1, (total+gap)/(minimum+gap))
	return min(max((total-(columns-1)*gap)/columns, minimum), maximum)
}

// Flow lays blocks out left-to-right, wrapping to a new row when the next block
// would overflow width, separated by gap spaces.
func Flow(width, gap int, blocks []string) string {
	if width <= 0 {
		return ""
	}
	gap = max(0, gap)
	spacer := strings.Repeat(" ", gap)
	var rows, row []string
	rowWidth := 0
	for _, block := range blocks {
		blockWidth := lipgloss.Width(block)
		added := blockWidth
		if len(row) > 0 {
			added += gap
		}
		if len(row) > 0 && rowWidth+added > width {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
			row, rowWidth = nil, 0
		}
		if len(row) > 0 && gap > 0 {
			row, rowWidth = append(row, spacer), rowWidth+gap
		}
		row, rowWidth = append(row, block), rowWidth+blockWidth
	}
	if len(row) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, row...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
