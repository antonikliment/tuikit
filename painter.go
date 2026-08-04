package tuikit

import (
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// Painter applies a style to text, or returns it untouched when the destination
// cannot take escapes.
//
// Passing one of these around is what stops every call site from repeating the
// "is this a terminal, and is NO_COLOR set" test — and from getting it subtly
// different, which is how half a command's output ends up colored.
type Painter func(lipgloss.Style, string) string

// Plain is the [Painter] that never paints. Tests use it to assert on content
// without decoding escape sequences.
func Plain(_ lipgloss.Style, text string) string { return text }

// Paint is the [Painter] that always paints, for output already known to be
// going to a terminal.
func Paint(s lipgloss.Style, text string) string { return s.Render(text) }

// PainterFor returns the [Painter] appropriate to w, deciding once so the
// answer cannot change halfway through rendering one command's output.
func PainterFor(w io.Writer) Painter {
	if !ColorEnabled(w) {
		return Plain
	}
	return Paint
}

// ColorEnabled reports whether output to w may carry ANSI escapes: w is a
// terminal and NO_COLOR is unset.
//
// It is a separate question from [IsTerminalWriter] on purpose. NO_COLOR says
// "do not paint", not "change format": a user who sets it still wants the
// readable table, just without the escapes. Conflating the two hands them the
// machine format, which is the opposite of what they asked for.
// See https://no-color.org.
func ColorEnabled(w io.Writer) bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	return IsTerminalWriter(w)
}

// IsTerminalWriter reports whether w is a terminal a human is watching. It is
// the predicate that decides human-vs-machine output.
//
// It asks about the output stream alone: rendering only writes, so a program
// reading nothing from stdin still has a human reading its stdout.
func IsTerminalWriter(w io.Writer) bool {
	file, ok := w.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(file.Fd())
}
