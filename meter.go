package tuikit

import (
	"image/color"

	"charm.land/bubbles/v2/progress"
)

// Meter is a fixed-width horizontal gauge (a filled/empty bar with no percentage
// label) over bubbles/progress. Use it for resource dials like CPU or memory.
type Meter struct {
	m progress.Model
}

// NewMeter builds a Meter width cells wide, filled in the given color.
func NewMeter(width int, fill color.Color) Meter {
	return Meter{m: progress.New(
		progress.WithWidth(width),
		progress.WithFillCharacters('█', '░'),
		progress.WithoutPercentage(),
		progress.WithColors(fill),
	)}
}

// View renders the meter at percent, clamped to 0..100.
func (g Meter) View(percent int) string {
	return g.m.ViewAs(float64(max(0, min(100, percent))) / 100)
}
