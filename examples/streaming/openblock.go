package main

import "strings"

// What the unsettled tail is currently inside.
//
// This is the "second parser" the note said a smarter reveal would need, kept
// as small as it can be: it classifies, it does not construct. Nothing here
// builds a tree or renders anything — it answers one question, which is how
// long the display is likely to wait before the next block settles.
//
// The kinds are ordered by how long they can plausibly stay open, because that
// is the only thing the pacing policy asks about.

type openKind int

const (
	openNothing   openKind = iota // between blocks; the next line can start anything
	openParagraph                 // prose, ends at the next blank line
	openContainer                 // list, quote, table, indented code, HTML block
	openFence                     // fenced code, ends only at a matching fence marker
)

func (k openKind) String() string {
	switch k {
	case openParagraph:
		return "paragraph"
	case openContainer:
		return "container"
	case openFence:
		return "fence"
	}
	return "nothing"
}

// openState is what is open and how much has arrived since it opened. The size
// matters as much as the kind: a fence twenty bytes in is an ordinary wait, and
// the same fence eight hundred bytes in is a display that has been frozen for
// four seconds.
type openState struct {
	kind openKind
	open int // bytes arrived since the construct opened
}

// classify reports what the tail is inside. It is given only the unsettled
// remainder, never the whole buffer: everything before the cut is by definition
// closed, so re-scanning it would be work with a known answer.
//
// The distinction that earns its keep is fence versus everything else. A
// paragraph or a list ends at the next blank line, which is rarely more than a
// line or two away. A fence ends only when a matching marker arrives, and
// nothing bounds how far away that is — which is the whole reason the display
// clock stalls.
func classify(tail string) openState {
	lines := strings.SplitAfter(tail, "\n")
	if last := len(lines) - 1; last >= 0 && lines[last] == "" {
		// The empty element SplitAfter leaves after a trailing newline is not a
		// blank line, and closing a construct on it would report the tail as
		// settled when its next line has simply not arrived yet.
		lines = lines[:last]
	}

	var (
		kind   openKind
		marker byte // the fence character in use, ` or ~
		length int  // and how long the opening run was
		opened int  // byte offset where the open construct started
		at     int  // bytes consumed
	)
	for _, line := range lines {
		body := strings.TrimSpace(line)
		start := at
		at += len(line)

		if kind == openFence {
			if closesFence(body, marker, length) {
				kind, opened = openNothing, at
			}
			continue
		}
		if char, run, ok := opensFence(body); ok {
			kind, marker, length, opened = openFence, char, run, start
			continue
		}

		switch {
		case body == "":
			kind, opened = openNothing, at
		case opensContainer(line, body):
			if kind != openContainer {
				opened = start
			}
			kind = openContainer
		case kind == openNothing:
			// A plain line with nothing open starts a paragraph. If a container
			// is already open this is a lazy continuation of it, so the kind is
			// left alone.
			kind, opened = openParagraph, start
		}
	}
	return openState{kind: kind, open: at - opened}
}

// opensFence reports whether body opens a fenced code block, and with what.
// The marker and its length are carried because a closing fence has to match
// both: "~~~" does not close "```", and "“" does not close "````".
func opensFence(body string) (marker byte, length int, ok bool) {
	if !strings.HasPrefix(body, "```") && !strings.HasPrefix(body, "~~~") {
		return 0, 0, false
	}
	marker = body[0]
	for length < len(body) && body[length] == marker {
		length++
	}
	return marker, length, true
}

// closesFence reports whether body is a closing fence for a run of length
// markers. A closing fence carries no info string, which is what keeps the
// opening line of one fence from being read as the close of another.
func closesFence(body string, marker byte, length int) bool {
	run := 0
	for run < len(body) && body[run] == marker {
		run++
	}
	return run >= length && strings.TrimSpace(body[run:]) == ""
}

// opensContainer reports whether line starts a block that runs until a blank
// line. These are the same constructs boundaryOf refuses to cut after, for the
// same reason — they keep absorbing lines — but here the question is how long
// that is likely to go on, not whether a cut is safe.
func opensContainer(line, body string) bool {
	switch {
	case strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t"):
		return true // indented code
	case strings.HasPrefix(body, ">"):
		return true // block quote
	case strings.Contains(body, "|"):
		return true // table row
	case strings.HasPrefix(body, "<"):
		return true // HTML block
	}
	return isBullet(body)
}

// isBullet reports whether body starts a bullet, numbered or task item.
func isBullet(body string) bool {
	for _, bullet := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(body, bullet) {
			return true
		}
	}
	digits := 0
	for digits < len(body) && body[digits] >= '0' && body[digits] <= '9' {
		digits++
	}
	return digits > 0 && digits < len(body)-1 &&
		(body[digits] == '.' || body[digits] == ')') && body[digits+1] == ' '
}
