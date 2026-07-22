package tuikit

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// TabStrip renders a row of labelled tab chips. The active tab is drawn as a
// three-sided box — border on left, top and right, open at the bottom — in its
// accent color, so it reads as a folder tab sitting on the panel below it. The
// rest are muted labels, bottom-aligned to the active tab's label row. Titles
// are pre-formatted by the caller (e.g. with counts). len(titles) must equal
// len(accents).
func (t Theme) TabStrip(titles []string, accents []color.Color, active int) string {
	parts := make([]string, 0, len(titles)*2)
	for i, title := range titles {
		if i > 0 {
			parts = append(parts, "  ")
		}
		if i == active {
			parts = append(parts, t.activeTab(accents[i]).Render(title))
		} else {
			parts = append(parts, t.inactiveChip().Render(title))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
}

// activeTab is the folder-tab style: a top/left/right border (no bottom) in the
// accent color, matching an accent panel rendered directly beneath it.
func (t Theme) activeTab(accent color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, true, false, true). // top, right, bottom, left
		BorderForeground(accent).
		Foreground(accent).
		Bold(true).
		Padding(0, 1)
}

func (t Theme) inactiveChip() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)
}
