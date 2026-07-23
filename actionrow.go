package tuikit

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// ActionRow renders a labelled row of selectable actions, e.g.
//
//	Actions:  Start  [Stop]  Restart
//
// The "Actions:" label is drawn in accent. When focused, the label at selected
// is bracketed and highlighted; the rest are muted. When not focused the whole
// row is muted, so a page can show which actions exist without implying the row
// is live.
func (t Theme) ActionRow(accent color.Color, selected int, labels []string, focused bool) string {
	if !focused {
		return t.MutedStyle().Render("Actions:  " + strings.Join(labels, "  "))
	}
	parts := []string{t.Accent(accent).Render("Actions:")}
	for i, label := range labels {
		if i == selected {
			parts = append(parts, t.selectedAction().Render("["+label+"]"))
		} else {
			parts = append(parts, t.MutedStyle().Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

// selectedAction highlights the chosen action distinctly from the accent label,
// so the selection stands out from the row's leading "Actions:" color.
func (t Theme) selectedAction() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Blue).Bold(true).Underline(true)
}
