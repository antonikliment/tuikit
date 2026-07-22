package tuikit

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// PanelStyle returns a bordered-panel style in the given accent. When focused it
// swaps to a double border in the theme's FocusBorder color.
func (t Theme) PanelStyle(accent color.Color, focused bool) lipgloss.Style {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(accent).Padding(0, 1)
	if focused {
		return style.Border(lipgloss.DoubleBorder()).BorderForeground(t.FocusBorder)
	}
	return style
}

// Panel is a convenience wrapper around PanelStyle with size baked in. Zero
// Width/Height means "fit content".
type Panel struct {
	Theme   Theme
	Accent  color.Color
	Focused bool
	Width   int
	Height  int
}

// Render draws content inside the panel.
func (p Panel) Render(content string) string {
	style := p.Theme.PanelStyle(p.Accent, p.Focused)
	if p.Width > 0 {
		style = style.Width(p.Width)
	}
	if p.Height > 0 {
		style = style.Height(p.Height)
	}
	return style.Render(content)
}
