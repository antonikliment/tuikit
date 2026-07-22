package tuikit

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// TabbedPanel renders a row of tabs joined seamlessly to a content panel: the
// active tab opens directly into the panel (its bottom edge is notched out of
// the panel's top border, so there is no dividing line), and both share the
// active tab's accent color. Inactive tabs are muted labels. width and height
// are the total footprint; body is the panel content.
func (t Theme) TabbedPanel(titles []string, accents []color.Color, active, width, height int, body string) string {
	width = max(width, 4)
	accent := accents[active]

	// Build the tab row, tracking the active tab's column offset and width so
	// the panel's top border can be notched to match.
	parts := make([]string, 0, len(titles)*2)
	activeLeft, activeWidth, col := 0, 0, 0
	for i, title := range titles {
		if i > 0 {
			parts = append(parts, "  ")
			col += 2
		}
		if i == active {
			chip := t.activeTab(accent).Render(title)
			activeLeft, activeWidth = col, lipgloss.Width(chip)
			parts = append(parts, chip)
			col += activeWidth
		} else {
			chip := t.inactiveChip().Render(title)
			parts = append(parts, chip)
			col += lipgloss.Width(chip)
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)

	topLine := t.Accent(accent).Render(notchedTop(width, activeLeft, activeWidth))

	// Panel body with left/right/bottom borders only — the notched top line is
	// its top edge.
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, true, true).
		BorderForeground(accent).
		Padding(0, 1).
		Width(width).
		Height(max(1, height-3)). // 2 tab-row lines + 1 top-border line
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, row, topLine, box)
}

// notchedTop builds a panel top-border line of the given total width with an
// opening (the "notch") of tabWidth cells starting at column left, so the tab
// above flows into the panel. The tab's left/right edges turn into the border
// with ┘ and └; the notch interior is blank.
func notchedTop(width, left, tabWidth int) string {
	if width < 2 {
		return strings.Repeat("─", max(0, width))
	}
	runes := make([]rune, width)
	for i := range runes {
		runes[i] = '─'
	}
	runes[0], runes[width-1] = '┌', '┐'

	right := min(left+tabWidth-1, width-1)
	if left <= 0 {
		runes[0] = '│' // active tab is flush left: its border continues down
	} else if left < width {
		runes[left] = '┘'
	}
	if right > 0 && right < width {
		runes[right] = '└'
	}
	for c := left + 1; c < right; c++ {
		if c > 0 && c < width {
			runes[c] = ' '
		}
	}
	return string(runes)
}
