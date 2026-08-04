package tuikit

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

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

// TransferredBytes renders transfer progress as "4.1 GiB / 6.6 GiB" using
// [FormatBytes].
//
// A total of zero or less means "not yet known" — a multi-file download does
// not learn its size until every manifest is fetched — and renders as the
// received count alone. That is deliberate: "4.1 GiB / 0 B" reads as a bug, and
// a caller with no total should show a byte counter rather than a progress bar
// pinned at zero.
func TransferredBytes(received, total int64) string {
	if total <= 0 {
		return FormatBytes(received)
	}
	return FormatBytes(received) + " / " + FormatBytes(total)
}

// CoarseDuration rounds d to roughly three significant figures, scaling the
// unit with the magnitude: 272.914939ms becomes 272ms while 1.234µs keeps its
// precision.
//
// Nanosecond precision on a wall-clock duration is noise in a column an
// operator is scanning for the one slow entry — 272.914939ms and 272ms lead to
// the same conclusion, and only one of them lines up. Durations under a
// microsecond are returned unchanged, as is the zero value. Negative durations
// round by magnitude and keep their sign.
func CoarseDuration(d time.Duration) time.Duration {
	for _, unit := range []time.Duration{time.Second, time.Millisecond, time.Microsecond} {
		if d.Abs() >= unit {
			return d.Round(unit / 100)
		}
	}
	return d
}

// AgeCutoff is how old an instant may be before [Age] stops rendering it as an
// elapsed time. Past a day "27h14m ago" is harder to place than the timestamp
// it came from.
const AgeCutoff = 24 * time.Hour

// Age renders at as how long before now it was: "3h23m0s ago". An operator
// reading a log or an event list wants "how long ago", and computing that from
// an RFC 3339 string in their head is the work this saves.
//
// Outside the window — an instant in the future, or older than [AgeCutoff] —
// the elapsed form stops being the useful one and the absolute local time comes
// back as RFC 3339. now is a parameter rather than a call to [time.Now] so the
// result is a pure function of its inputs and a test need not sleep.
func Age(at, now time.Time) string {
	since := now.Sub(at).Round(time.Second)
	if since < 0 || since > AgeCutoff {
		return at.Local().Format(time.RFC3339)
	}
	return since.String() + " ago"
}

// Titleize turns an identifier into a label: "running_profiles" and its camel
// spelling "runningProfiles" both become "Running profiles". Sentence case, not
// Title Case, so "Autostart profiles" does not read as a proper noun.
func Titleize(name string) string {
	var b strings.Builder
	for i, r := range name {
		switch {
		case r == '_':
			b.WriteRune(' ')
		case unicode.IsUpper(r) && i > 0:
			b.WriteRune(' ')
			b.WriteRune(unicode.ToLower(r))
		case i == 0:
			b.WriteRune(unicode.ToUpper(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// healthy names the status words that mean "nothing to look at here".
// Everything unrecognized is a warning, so an operator scanning output only has
// to read the words that are not green.
var healthy = map[string]bool{
	"ok": true, "running": true, "ready": true, "completed": true,
	"installed": true, "active": true, "healthy": true, "done": true,
	"enabled":   true,
	"succeeded": true, "success": true, "passed": true,
}

// ClassifyStatus grades a status word so it can be colored without every caller
// keeping its own list of what "fine" looks like. Matching is case-insensitive;
// anything unrecognized is [LevelWarning], which is the safe default — an
// unfamiliar state is one an operator should read.
//
// An empty string is [LevelInfo]: there is nothing to grade.
func ClassifyStatus(word string) Level {
	key := strings.ToLower(word)
	switch {
	case key == "":
		return LevelInfo
	case healthy[key]:
		return LevelSuccess
	case strings.Contains(key, "fail") || strings.Contains(key, "error"):
		return LevelError
	default:
		return LevelWarning
	}
}
