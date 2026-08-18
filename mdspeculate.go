package tuikit

import "strings"

// Closing the constructs a partial buffer leaves open, so the unsettled tail
// can be rendered rather than shown raw.
//
// boundaryOf answers where the buffer may be cut. Everything after that cut is
// mid-construct by definition, which is why it is normally wrapped verbatim:
// handed to a markdown engine as-is, "a **bold" renders the asterisks
// literally, because the emphasis never closes. The fix is the one bracket
// matchers have always used — keep a stack of what is open, and append the
// closers in reverse before rendering. The synthetic closers are thrown away
// again next frame, when more text has arrived and the stack is recomputed.
//
// This is a display convenience and nothing more. It runs only on the tail,
// which is redrawn every frame and never cached, so a wrong guess is corrected
// within one frame and can never corrupt output that has already settled. That
// is what lets the scan below be an approximation of CommonMark's inline rules
// rather than an implementation of them: full fidelity needs delimiter-run
// flanking analysis, and the difference only shows in cases that are about to
// be re-rendered anyway.
//
// Fences are deliberately absent from the stack. CommonMark closes an open
// fence at the end of the document, so a partial fence already renders as a
// code block — and, through a highlighting renderer, as *highlighted* code.
// A synthetic "```" would be a no-op. This is the one construct the display
// clock cannot stream (nothing settles until the fence closes) and the one
// this approach gets for free.

// closeOpen returns tail with synthetic closers appended for the inline
// constructs it leaves open, ready to be rendered as markdown.
//
// A trailing opener with nothing after it is truncated instead of closed. "a *"
// is far more likely to be the first character of "**" than an emphasis span
// the user opened and abandoned, and closing it produces a literal "**" that
// flickers away next frame.
func closeOpen(tail string) string {
	stack := scanOpen(tail)
	if len(stack) == 0 {
		return tail
	}

	// The topmost entry is the innermost, so it is the only one that can be
	// touching the end of the buffer.
	if top := stack[len(stack)-1]; strings.TrimSpace(tail[top.end:]) == "" {
		// Trailing spaces go with it: they were separating text from a
		// delimiter that is no longer there.
		tail = strings.TrimRight(tail[:top.start], " \t")
		stack = stack[:len(stack)-1]
	}

	var b strings.Builder
	b.WriteString(tail)
	for i := len(stack) - 1; i >= 0; i-- {
		b.WriteString(stack[i].closer)
	}
	return b.String()
}

// open is one unclosed construct: what would close it, and where it was opened,
// so a trailing opener can be recognized and dropped.
type open struct {
	closer     string
	start, end int
}

// scanOpen walks tail and returns what is still open, outermost first.
//
// The scan is line-oriented because two block-level facts suppress inline
// parsing entirely — a fence, and an indented code block — and a blank line
// ends a paragraph, which abandons any inline construct still open inside it.
func scanOpen(tail string) []open {
	var (
		stack  []open
		fenced bool
		marker byte
		run    int
	)

	for offset := 0; offset < len(tail); {
		end := strings.IndexByte(tail[offset:], '\n')
		line := tail[offset:]
		if end >= 0 {
			line = tail[offset : offset+end]
		}
		body := strings.TrimSpace(line)

		switch {
		case fenced:
			if closesFence(body, marker, run) {
				fenced = false
			}
		case isFence(body):
			// Nothing inside a fence is inline markdown, and the fence itself
			// needs no closer: CommonMark ends it at the end of the document.
			fenced, marker, run = true, body[0], fenceRun(body)
		case body == "":
			// A blank line ends the paragraph, and with it anything left open
			// inside it. Emphasis does not span blocks.
			stack = stack[:0]
		case strings.HasPrefix(line, "    ") && len(stack) == 0:
			// An indented code block, but only where a paragraph is not already
			// open — otherwise this is a lazy continuation line of that
			// paragraph, which is ordinary inline text.
		default:
			stack = scanInline(tail, offset, offset+len(line), stack)
		}

		if end < 0 {
			break
		}
		offset += end + 1
	}
	return stack
}

// scanInline walks one line of inline text, pushing and popping the delimiter
// stack. The stack is carried between lines: a paragraph is one inline context,
// and "**bold" on one line closes on the next.
func scanInline(text string, offset, limit int, stack []open) []open {
	for i := offset; i < limit; {
		switch c := text[i]; c {
		case '\\':
			i += 2
			continue

		case '`', '*', '_', '~':
			// A run of the same character is one delimiter. Matching runs by
			// length is what keeps "**" and "*" apart without implementing
			// flanking rules.
			start := i
			for i < limit && text[i] == c {
				i++
			}
			delim := text[start:i]
			if c == '~' && len(delim) < 2 {
				continue // a lone tilde is literal
			}
			if top := len(stack) - 1; top >= 0 && stack[top].closer == delim {
				stack = stack[:top]
				continue
			}
			if inCode(stack) && c != '`' {
				continue // emphasis is not parsed inside a code span
			}
			stack = append(stack, open{closer: delim, start: start, end: i})
			continue

		case '[':
			if !inCode(stack) {
				stack = append(stack, open{closer: "]", start: i, end: i + 1})
			}

		case ']':
			if top := len(stack) - 1; top >= 0 && stack[top].closer == "]" {
				stack = stack[:top]
				// A link is "](dest)", so the bracket closing hands straight
				// over to the parenthesis.
				if i+1 < limit && text[i+1] == '(' {
					stack = append(stack, open{closer: ")", start: i + 1, end: i + 2})
					i += 2
					continue
				}
			}

		case ')':
			if top := len(stack) - 1; top >= 0 && stack[top].closer == ")" {
				stack = stack[:top]
			}
		}
		i++
	}
	return stack
}

// inCode reports whether a code span is open. Everything inside one is literal,
// so no other delimiter may be pushed while it is.
func inCode(stack []open) bool {
	for _, o := range stack {
		if strings.HasPrefix(o.closer, "`") {
			return true
		}
	}
	return false
}

// closesFence reports whether body is a closing fence for a block opened with
// run occurrences of marker. A closing fence carries no info string and must be
// at least as long as the opening one.
func closesFence(body string, marker byte, run int) bool {
	return len(body) > 0 && body[0] == marker && fenceRun(body) >= run &&
		strings.TrimRight(body, string(marker)) == ""
}

func fenceRun(body string) int {
	n := 0
	for n < len(body) && body[n] == body[0] {
		n++
	}
	return n
}
