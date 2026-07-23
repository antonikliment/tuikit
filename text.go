package tuikit

import "fmt"

// TruncMiddle keeps a string on a single line by eliding its middle with an
// ellipsis once it exceeds width runes, so long paths never orphan onto a
// wrapped line beneath their label. It is rune-aware; width counts runes.
func TruncMiddle(s string, width int) string {
	runes := []rune(s)
	if width <= 1 || len(runes) <= width {
		return s
	}
	head := (width - 1) / 2
	return string(runes[:head]) + "…" + string(runes[len(runes)-(width-1-head):])
}

// FormatBytes renders a byte count as a human-readable IEC size (B, KiB, MiB,
// GiB, TiB, PiB) with one decimal place. It carries no external dependency so
// tuikit stays dependency-light.
func FormatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	value, suffixes := float64(size), []string{"KiB", "MiB", "GiB", "TiB"}
	for _, suffix := range suffixes {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f PiB", value/unit)
}
