package tuikit

import "strings"

// Where a partial markdown buffer may be cut so the part before it renders the
// same as it will in the finished document.
//
// Markdown is not a line-at-a-time format. A block keeps absorbing lines until
// something ends it, and several constructs reach backwards: "===" under a
// paragraph turns it into a heading, a blank line inside a list is a loose-list
// marker rather than a terminator, and a link reference defined at the bottom
// of a document decides how a link at the top is drawn. Cutting mid-construct
// therefore does not merely truncate — it changes how already-rendered text is
// interpreted.
//
// These rules follow CommonMark's block-boundary semantics. Charm's crush
// arrived at an equivalent set for its chat view; this is an independent
// implementation of the same published behavior.

// boundaryOf returns the byte offset of the last position in text that is
// provably a block boundary, or 0 when no part of the buffer has settled.
//
// It is deliberately pessimistic: every rule below rejects a candidate cut that
// *might* be mid-construct, because a wrong cut corrupts output that has
// already been shown, while a missed one only delays formatting by a frame.
//
// The scan is a single forward pass. It runs once a frame on the whole buffer,
// so anything super-linear here shows up as the document grows — an earlier
// version re-scanned the settled prefix for link-reference definitions at every
// candidate blank line, which cost 49ms on a 74KB buffer.
func boundaryOf(text string) int {
	lines := strings.SplitAfter(text, "\n")
	if last := len(lines) - 1; last >= 0 && lines[last] == "" {
		// SplitAfter leaves an empty element after a trailing newline. It is
		// not a blank line, and treating it as one would cut at the end of a
		// buffer whose last line may still be growing.
		lines = lines[:last]
	}

	var (
		offset  int    // bytes consumed, and so the offset a cut here would have
		best    int    // last confirmed cut
		pending int    // a candidate cut still waiting on the line that follows it
		last    string // last non-blank line seen, blank lines inside a fence aside
		fenced  bool
		linkRef bool // a definition has been seen, so nothing after it can settle
	)
	for _, line := range lines {
		body := strings.TrimSpace(line)
		offset += len(line)

		// Candidate cuts are blank lines outside a fence: the one separator
		// markdown uses to end nearly every block. A candidate is held rather
		// than taken, because a "===" on the far side would retroactively
		// promote the paragraph above it into a heading — so the cut is not
		// confirmed until a following line proves it is not a setext underline.
		if body == "" && !fenced {
			if !linkRef && !opensConstruct(last) {
				pending = offset
			}
			continue
		}

		if pending > 0 {
			if !isSetextUnderline(body) {
				best = pending
			}
			pending = 0
		}
		if isFence(body) {
			fenced = !fenced
		}
		if isLinkReference(body) {
			// Unlike every other construct, a definition is not local: it
			// resolves links anywhere in the document, so it and its uses must
			// be rendered together. Once one appears no later cut is safe, since
			// the tail would be rendered without it and draw the link
			// unresolved.
			linkRef = true
		}
		if body != "" {
			last = line
		}
	}
	if pending > 0 {
		// Nothing follows the last candidate, so there is no setext underline
		// waiting on the far side of it to hold it back.
		best = pending
	}
	return best
}

// opensConstruct reports whether line starts something the next lines continue.
func opensConstruct(line string) bool {
	body := strings.TrimSpace(line)
	switch {
	case body == "":
		return false
	case strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t"):
		return true // indented code block
	case strings.HasPrefix(body, ">"):
		return true // block quote
	case strings.Contains(body, "|"):
		return true // table row, possibly awaiting its delimiter row
	case strings.HasPrefix(body, "<"):
		return true // HTML block, which runs to its own terminator
	case isListItem(body):
		return true
	}
	// Any plain paragraph line is a setext heading until the next line proves
	// otherwise, so a paragraph is only safe to cut after when we have seen
	// what follows it — which firstContentLine, above, checks.
	return false
}

// isSetextUnderline reports whether line is a "===" or "---" rule that would
// turn the paragraph above it into a heading.
func isSetextUnderline(line string) bool {
	body := strings.TrimSpace(line)
	if body == "" {
		return false
	}
	return strings.Trim(body, "=") == "" || strings.Trim(body, "-") == ""
}

// isFence reports whether body opens or closes a fenced code block.
func isFence(body string) bool {
	return strings.HasPrefix(body, "```") || strings.HasPrefix(body, "~~~")
}

// isListItem reports whether body starts a bullet, numbered, or task item.
func isListItem(body string) bool {
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

// isLinkReference reports whether body is a "[label]: url" definition.
func isLinkReference(body string) bool {
	if !strings.HasPrefix(body, "[") {
		return false
	}
	return strings.Index(body, "]:") > 0
}
