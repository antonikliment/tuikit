package tuikit

import (
	"fmt"
	"strings"
	"testing"
)

// countItem is a ListItem that records how often it was actually rendered,
// which is how the memo and the windowing are observed — both are otherwise
// invisible from outside.
type countItem struct {
	id     string
	lines  int
	rev    int
	body   string
	render *int
}

func (i *countItem) ID() string { return i.id }

func (i *countItem) Revision() int { return i.rev }

func (i *countItem) Render(width int) string {
	*i.render++
	body := i.body
	if body == "" {
		body = i.id
	}
	out := make([]string, i.lines)
	for n := range out {
		out[n] = fmt.Sprintf("%s:%d w%d %s", i.id, n, width, body)
	}
	return strings.Join(out, "\n")
}

// items builds n single-line items sharing one render counter.
func items(n int, calls *int) []ListItem {
	out := make([]ListItem, n)
	for i := range out {
		out[i] = &countItem{id: fmt.Sprintf("i%d", i), lines: 1, render: calls}
	}
	return out
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// The headline claim: a huge list costs a viewport, not a transcript.
func TestRenderOnlyRendersTheVisibleWindow(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(5000, &calls))

	got := l.Render(40, 40)
	if calls > 40 {
		t.Fatalf("rendered %d of 5000 items for a 40-line viewport, want at most 40", calls)
	}
	if n := lineCount(got); n != 40 {
		t.Fatalf("rendered %d lines, want 40", n)
	}
	if !strings.HasSuffix(got, "i4999:0 w40 i4999") {
		t.Fatalf("tail not visible, last line of:\n%s", got)
	}

	before := calls
	if l.Render(40, 40) != got {
		t.Fatal("second frame differs from the first")
	}
	if calls != before {
		t.Fatalf("a repeated frame rendered %d more items, want 0", calls-before)
	}
}

func TestAppendingWhileAtBottomKeepsTheTailVisible(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)

	before := calls
	l.Append(&countItem{id: "new", lines: 1, render: &calls})
	got := l.Render(40, 10)

	if !strings.HasSuffix(got, "new:0 w40 new") {
		t.Fatalf("appended item is not at the tail:\n%s", got)
	}
	if calls-before != 1 {
		t.Fatalf("appending rendered %d items, want 1", calls-before)
	}
}

func TestAppendingWhileScrolledUpDoesNotMoveTheView(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)
	l.ScrollBy(-30)
	if l.Following() {
		t.Fatal("scrolling up left the list following the tail")
	}
	want := l.Render(40, 10)

	l.Append(&countItem{id: "new", lines: 1, render: &calls})
	if got := l.Render(40, 10); got != want {
		t.Fatalf("append moved the view:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// Prepending history must not jump the view either — the anchor is an ID, not
// an index.
func TestPrependingHistoryKeepsTheAnchorInPlace(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)
	l.ScrollBy(-40)
	want := l.Render(40, 10)

	history := items(50, &calls)
	for i := range history {
		history[i].(*countItem).id = "h" + history[i].ID()
	}
	l.SetItems(append(history, l.items...))

	if got := l.Render(40, 10); got != want {
		t.Fatalf("prepend moved the view:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestScrollingDownToTheBottomResumesFollowing(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)
	l.ScrollBy(-20)
	l.ScrollBy(5)
	if l.Following() {
		t.Fatal("a partial scroll down resumed following early")
	}
	l.ScrollBy(50)
	if !l.Following() {
		t.Fatal("scrolling past the end did not resume following")
	}
	if !strings.HasSuffix(l.Render(40, 10), "i99:0 w40 i99") {
		t.Fatal("following the tail did not show the last item")
	}
}

func TestInvalidateRerendersExactlyThatItem(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)

	before := calls
	l.Invalidate("i99")
	l.Render(40, 10)
	if calls-before != 1 {
		t.Fatalf("invalidating one item rendered %d items, want 1", calls-before)
	}
}

// The streaming case without an explicit Invalidate: the growing tail reports a
// new revision and nothing above it is touched.
func TestARevisionBumpRerendersOnlyTheChangedItem(t *testing.T) {
	calls := 0
	tail := &countItem{id: "tail", lines: 1, render: &calls}
	l := NewMemoList()
	l.SetItems(append(items(100, &calls), tail))
	l.Render(40, 10)

	before := calls
	tail.rev, tail.body = 1, "more text"
	got := l.Render(40, 10)
	if calls-before != 1 {
		t.Fatalf("a revision bump rendered %d items, want 1", calls-before)
	}
	if !strings.HasSuffix(got, "more text") {
		t.Fatalf("the new revision is not on screen:\n%s", got)
	}
}

func TestWidthChangeInvalidatesEveryCachedRender(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)

	before := calls
	got := l.Render(60, 10)
	if calls-before < 10 {
		t.Fatalf("a width change rendered %d items, want the visible window re-rendered", calls-before)
	}
	if !strings.Contains(got, "w60") {
		t.Fatalf("the new width is not in the output:\n%s", got)
	}
}

// A width change while scrolled up must re-anchor rather than run off the end
// of the newly-taller content.
func TestWidthChangeReanchorsWhileScrolledUp(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems([]ListItem{
		&countItem{id: "a", lines: 5, render: &calls},
		&countItem{id: "b", lines: 5, render: &calls},
		&countItem{id: "c", lines: 5, render: &calls},
	})
	l.Render(40, 6)
	l.ScrollBy(-9)
	if got := lineCount(l.Render(80, 6)); got != 6 {
		t.Fatalf("rendered %d lines after a resize, want 6", got)
	}
}

func TestRenderHandlesTheExtremes(t *testing.T) {
	calls := 0
	l := NewMemoList()
	if got := l.Render(40, 10); got != "" {
		t.Fatalf("empty list rendered %q, want %q", got, "")
	}
	l.ScrollBy(-5) // no viewport yet: must not panic
	l.ScrollToTop()
	l.ScrollToBottom()

	l.SetItems(items(3, &calls))
	if got := lineCount(l.Render(40, 40)); got != 3 {
		t.Fatalf("a viewport taller than the content rendered %d lines, want 3", got)
	}
	l.ScrollBy(-100)
	if got := lineCount(l.Render(40, 40)); got != 3 {
		t.Fatalf("scrolling past the top rendered %d lines, want 3", got)
	}
	l.ScrollBy(100)
	if got := lineCount(l.Render(40, 40)); got != 3 {
		t.Fatalf("scrolling past the bottom rendered %d lines, want 3", got)
	}
	if got := l.Render(0, 10); got != "" {
		t.Fatalf("zero width rendered %q, want %q", got, "")
	}
	if got := l.Render(40, 0); got != "" {
		t.Fatalf("zero height rendered %q, want %q", got, "")
	}
}

func TestScrollToTopShowsTheFirstItem(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(100, &calls))
	l.Render(40, 10)
	l.ScrollToTop()
	got := l.Render(40, 10)
	if !strings.HasPrefix(got, "i0:0 w40 i0") {
		t.Fatalf("ScrollToTop did not show the first item:\n%s", got)
	}
	if l.Following() {
		t.Fatal("ScrollToTop left the list following the tail")
	}
}

// Multi-line items must be shown partially at the top edge rather than being
// skipped or forced whole into the window.
func TestATallItemIsClippedAtTheTopEdge(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems([]ListItem{
		&countItem{id: "a", lines: 10, render: &calls},
		&countItem{id: "b", lines: 10, render: &calls},
	})
	l.Render(40, 4)
	l.ScrollBy(-10)
	want := "a:6 w40 a\na:7 w40 a\na:8 w40 a\na:9 w40 a"
	if got := l.Render(40, 4); got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
	if l.Len() != 2 {
		t.Fatalf("Len = %d, want 2", l.Len())
	}
}

func TestStaleCacheEntriesAreDropped(t *testing.T) {
	calls := 0
	l := NewMemoList()
	l.SetItems(items(200, &calls))
	l.ScrollToTop()
	for range 200 {
		l.Render(40, 10)
		l.ScrollBy(1)
	}
	if len(l.memo) < 100 {
		t.Fatalf("cached %d entries after walking the list, want the walked items cached", len(l.memo))
	}
	l.SetItems(items(2, &calls))
	if len(l.memo) > 66 {
		t.Fatalf("kept %d cache entries for a 2-item list, want the stale ones dropped", len(l.memo))
	}
}
