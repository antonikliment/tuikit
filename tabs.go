package tuikit

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// TabStrip renders a row of labelled tab chips, drawing the active one as a
// filled accent chip and the rest as muted labels. Titles are pre-formatted by
// the caller (e.g. with counts). len(titles) must equal len(accents).
func (t Theme) TabStrip(titles []string, accents []color.Color, active int) string {
	chips := make([]string, len(titles))
	for i, title := range titles {
		if i == active {
			chips[i] = t.activeChip(accents[i]).Render(title)
		} else {
			chips[i] = t.inactiveChip().Render(title)
		}
	}
	return strings.Join(chips, " ")
}

func (t Theme) activeChip(accent color.Color) lipgloss.Style {
	return lipgloss.NewStyle().Background(accent).Foreground(t.TabActiveFg).Bold(true).Padding(0, 1)
}

func (t Theme) inactiveChip() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted).Padding(0, 1)
}
