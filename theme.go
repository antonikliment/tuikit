package tuikit

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is the configurable palette every tuikit component draws from. Swap the
// colors (or build one from scratch) to reskin the whole kit; nothing else in
// the library hardcodes a color.
type Theme struct {
	Green  color.Color
	Blue   color.Color
	Yellow color.Color
	Red    color.Color
	Cyan   color.Color
	Muted  color.Color
	Brand  color.Color

	// TabActiveFg is the foreground drawn on a filled active tab chip.
	TabActiveFg color.Color
	// FocusBorder is the border color of a focused Panel.
	FocusBorder color.Color
}

// DefaultTheme returns the stock 16-color-safe palette.
func DefaultTheme() Theme {
	return Theme{
		Green:       lipgloss.Color("10"),
		Blue:        lipgloss.Color("12"),
		Yellow:      lipgloss.Color("11"),
		Red:         lipgloss.Color("9"),
		Cyan:        lipgloss.Color("14"),
		Muted:       lipgloss.Color("8"),
		Brand:       lipgloss.Color("63"),
		TabActiveFg: lipgloss.Color("0"),
		FocusBorder: lipgloss.Color("11"),
	}
}

// MutedStyle renders de-emphasized text.
func (t Theme) MutedStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(t.Muted) }

// SubtleStyle is MutedStyle, dimmer still — for footer/status chrome.
func (t Theme) SubtleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted).Faint(true)
}

// BrandStyle renders the app/brand name.
func (t Theme) BrandStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Brand).Bold(true)
}

// Accent renders text in the given accent color.
func (t Theme) Accent(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
