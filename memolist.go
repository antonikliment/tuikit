package tuikit

import "strings"

// ListItem is one block of a [MemoList]: a stable identity and a full render at
// a given width. Render may return many lines — a chat message, a tool block, a
// diff — and the list treats it as one indivisible unit for caching, though it
// will show a partial one at the edges of the viewport.
type ListItem interface {
	// ID identifies the item across frames. It is the memo key, so two items
	// sharing an ID share a cache entry: give every item its own.
	ID() string
	// Render lays the item out at width. It is called only when the item is
	// about to be measured or drawn and its cache entry is missing.
	Render(width int) string
}

// RevisedItem is the optional half of [ListItem]: an item whose content changes
// under a fixed ID reports a revision that changes with it, and the memo drops
// the stale entry on its own. It is the declarative alternative to calling
// [MemoList.Invalidate] — useful when the item is a view onto data the caller
// mutates elsewhere. Items that do not implement it are treated as revision 0,
// which is to say immutable until invalidated.
type RevisedItem interface {
	Revision() int
}

// memoEntry is one item's cached render: the revision it was built at, and the
// lines it produced. The line count is the height cache — heights and rendered
// text share an entry, so they can never disagree and cannot be invalidated
// separately.
type memoEntry struct {
	rev   int
	lines []string
}

// anchor is the item and the line inside it drawn at the top of the viewport.
// Anchoring by ID rather than by index is what keeps the view still when
// history is prepended: the item under the top edge is still the same item, so
// it stays where it is.
type anchor struct {
	id   string
	line int
}

// MemoList is a virtualized list for long scrollback — a chat transcript, a log
// pane — that renders only what is on screen and memoizes each item's rendered
// block. A view that rebuilds its whole content every frame costs O(transcript)
// per repaint, which is why long agent sessions in terminal chat UIs get slower
// as they get longer; here scrolling is a walk over cached lines and appending
// costs one item's render.
//
// The list owns no colors and no chrome: items render themselves, so a caller
// composes a [Panel] or a footer around the result. It also owns no input —
// pass scroll deltas to [MemoList.ScrollBy] from whatever keys or wheel events
// the host binds, keeping it free of any framework. It runs no goroutines.
//
// The zero value is not usable; build one with [NewMemoList].
type MemoList struct {
	items []ListItem
	memo  map[string]memoEntry

	// Geometry of the last Render, which is what ScrollBy does its arithmetic
	// against — a scroll before the first frame has no viewport to move.
	width  int
	height int

	anchor anchor
	follow bool
}

// NewMemoList returns an empty MemoList following the tail.
func NewMemoList() MemoList {
	return MemoList{memo: map[string]memoEntry{}, follow: true}
}

// SetItems replaces the backing items. Cache entries survive by ID, so
// reordering, prepending history, or replacing the slice with a superset of
// itself costs no renders; entries for items that have gone away are dropped
// once enough of them accumulate.
func (l *MemoList) SetItems(items []ListItem) {
	l.items = items
	l.prune()
}

// Append adds items at the tail — the common case, and the cheap one: nothing
// already cached is touched, so a frame that appends one message renders one
// message.
func (l *MemoList) Append(items ...ListItem) {
	l.items = append(l.items, items...)
}

// Len is the number of items.
func (l *MemoList) Len() int { return len(l.items) }

// Invalidate drops one item's cached render, so the next frame rebuilds exactly
// that item. This is the streaming case: the last item grows every frame and
// nothing above it does.
func (l *MemoList) Invalidate(id string) { delete(l.memo, id) }

// InvalidateAll drops every cached render — for a theme swap, or any change to
// how items draw that the items themselves do not report.
func (l *MemoList) InvalidateAll() { clear(l.memo) }

// Following reports whether the list is stuck to the tail.
func (l *MemoList) Following() bool { return l.follow }

// ScrollToBottom returns to the tail and resumes following it.
func (l *MemoList) ScrollToBottom() { l.follow = true }

// ScrollToTop jumps to the first item and stops following.
func (l *MemoList) ScrollToTop() {
	l.follow = false
	if len(l.items) > 0 {
		l.anchor = anchor{id: l.items[0].ID()}
	}
}

// ScrollBy moves the view by delta lines: negative up, positive down. Scrolling
// up stops following the tail, and scrolling down resumes it on arrival at the
// bottom — the universal chat behavior.
//
// It is a no-op before the first [MemoList.Render], which is where the viewport
// size comes from.
func (l *MemoList) ScrollBy(delta int) {
	if delta == 0 || l.width <= 0 || l.height <= 0 || len(l.items) == 0 {
		return
	}
	idx, line := l.moveAnchor(delta)
	l.anchor = anchor{id: l.items[idx].ID(), line: line}
	l.follow = delta > 0 && l.linesBelow(l.height+1) <= l.height
}

// Render lays out the visible window: width columns, at most height lines.
// Items outside the window are never rendered, and items inside it are rendered
// only on a cache miss, so a steady frame over a settled transcript does no
// work beyond joining cached lines.
//
// A width change invalidates everything, since wrapping changes every height.
// An empty list renders "", and a viewport taller than the content renders just
// the content.
func (l *MemoList) Render(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if width != l.width {
		clear(l.memo)
		l.width = width
	}
	l.height = height
	if len(l.items) == 0 {
		return ""
	}
	// Falling back to the tail when the anchor no longer has a viewport's worth
	// of content below it is what keeps the bottom edge pinned after a resize,
	// a truncation, or a scroll that overshot the end.
	if l.follow || l.linesBelow(height+1) <= height {
		return strings.Join(l.tailWindow(height), "\n")
	}
	return strings.Join(l.anchoredWindow(height), "\n")
}

// anchoredWindow collects height lines downward from the anchor.
func (l *MemoList) anchoredWindow(height int) []string {
	idx, line := l.anchorPos()
	out := make([]string, 0, height)
	for i := idx; i < len(l.items) && len(out) < height; i++ {
		lines := l.lines(i)
		if i == idx && line > 0 {
			if line >= len(lines) {
				continue
			}
			lines = lines[line:]
		}
		out = append(out, lines...)
	}
	if len(out) > height {
		out = out[:height]
	}
	return out
}

// tailWindow collects the last height lines, walking backwards from the last
// item and measuring only the items it reaches. It re-anchors as it goes, so a
// scroll away from the tail starts from where the tail actually was.
func (l *MemoList) tailWindow(height int) []string {
	var out []string
	idx := len(l.items) - 1
	for ; idx >= 0 && len(out) < height; idx-- {
		out = append(append([]string{}, l.lines(idx)...), out...)
	}
	idx++
	over := max(len(out)-height, 0)
	l.anchor = anchor{id: l.items[idx].ID(), line: over}
	return out[over:]
}

// moveAnchor walks the anchor delta lines through the items, measuring only
// the ones it crosses, and returns the item index and line it lands on. The
// bottom end is left to Render's tail fallback rather than clamped here, so the
// two do not have to agree about what "the end" is.
func (l *MemoList) moveAnchor(delta int) (int, int) {
	idx, line := l.anchorPos()
	line += delta
	for line < 0 && idx > 0 {
		idx--
		line += len(l.lines(idx))
	}
	for idx < len(l.items)-1 && line >= len(l.lines(idx)) {
		line -= len(l.lines(idx))
		idx++
	}
	return idx, min(max(line, 0), len(l.lines(idx))-1)
}

// anchorPos resolves the anchor to an item index and a line inside it, falling
// back to the top when the anchored item has gone away.
func (l *MemoList) anchorPos() (int, int) {
	for i, item := range l.items {
		if item.ID() == l.anchor.id {
			return i, max(l.anchor.line, 0)
		}
	}
	return 0, 0
}

// linesBelow counts the lines from the anchor to the end, stopping at limit so
// a long transcript costs a few measurements rather than all of them.
func (l *MemoList) linesBelow(limit int) int {
	idx, line := l.anchorPos()
	n := -line
	for i := idx; i < len(l.items) && n < limit; i++ {
		n += len(l.lines(i))
	}
	return max(n, 0)
}

// lines returns item i's rendered lines, rendering it only on a cache miss.
func (l *MemoList) lines(i int) []string {
	item := l.items[i]
	id, rev := item.ID(), revisionOf(item)
	if entry, ok := l.memo[id]; ok && entry.rev == rev {
		return entry.lines
	}
	lines := strings.Split(strings.TrimRight(item.Render(l.width), "\n"), "\n")
	l.memo[id] = memoEntry{rev: rev, lines: lines}
	return lines
}

// prune drops cache entries whose items are gone, once they outnumber the live
// ones by enough to be worth a sweep. Scanning on every SetItems would make the
// cheap path — replacing the slice with a longer version of itself — walk the
// whole transcript every frame.
func (l *MemoList) prune() {
	if len(l.memo) <= len(l.items)+64 {
		return
	}
	live := make(map[string]struct{}, len(l.items))
	for _, item := range l.items {
		live[item.ID()] = struct{}{}
	}
	for id := range l.memo {
		if _, ok := live[id]; !ok {
			delete(l.memo, id)
		}
	}
}

// revisionOf is an item's revision, or 0 for the immutable majority.
func revisionOf(item ListItem) int {
	if r, ok := item.(RevisedItem); ok {
		return r.Revision()
	}
	return 0
}
