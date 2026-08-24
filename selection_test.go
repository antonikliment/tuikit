package tuikit

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const selectionView = "hello world\nsecond line\nthird line"

func highlight() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color("#1c1830"))
}

func TestSelectionEmptyUntilDragged(t *testing.T) {
	var s Selection
	if !s.Empty() {
		t.Fatal("zero selection is not empty")
	}
	s.Begin(Cell{Row: 0, Col: 2})
	if !s.Empty() {
		t.Fatal("a click without motion selects something")
	}
	s.Extend(Cell{Row: 0, Col: 5})
	if s.Empty() {
		t.Fatal("a drag selects nothing")
	}
	s.Clear()
	if !s.Empty() {
		t.Fatal("Clear left a selection behind")
	}
}

func TestSelectionExtendBeforeBeginDoesNothing(t *testing.T) {
	var s Selection
	s.Extend(Cell{Row: 1, Col: 4})
	if !s.Empty() || s.Text(selectionView) != "" {
		t.Fatalf("Extend without Begin selected %q", s.Text(selectionView))
	}
}

func TestSelectionTextSingleLine(t *testing.T) {
	var s Selection
	s.Begin(Cell{Row: 0, Col: 6})
	s.Extend(Cell{Row: 0, Col: 11})
	if got := s.Text(selectionView); got != "world" {
		t.Fatalf("Text = %q, want %q", got, "world")
	}
}

func TestSelectionTextMultiLine(t *testing.T) {
	var s Selection
	s.Begin(Cell{Row: 0, Col: 6})
	s.Extend(Cell{Row: 2, Col: 5})
	want := "world\nsecond line\nthird"
	if got := s.Text(selectionView); got != want {
		t.Fatalf("Text = %q, want %q", got, want)
	}
}

func TestSelectionTextIsTheSameDraggedBackwards(t *testing.T) {
	var forward, backward Selection
	forward.Begin(Cell{Row: 0, Col: 6})
	forward.Extend(Cell{Row: 2, Col: 5})
	backward.Begin(Cell{Row: 2, Col: 5})
	backward.Extend(Cell{Row: 0, Col: 6})
	if got, want := backward.Text(selectionView), forward.Text(selectionView); got != want {
		t.Fatalf("backward drag = %q, forward = %q", got, want)
	}
}

func TestSelectionIgnoresRowsAndColumnsOffTheFrame(t *testing.T) {
	var s Selection
	s.Begin(Cell{Row: 1, Col: 0})
	s.Extend(Cell{Row: 40, Col: 200})
	want := "second line\nthird line"
	if got := s.Text(selectionView); got != want {
		t.Fatalf("Text = %q, want %q", got, want)
	}
}

func TestSelectionTextTrimsSelectedPadding(t *testing.T) {
	var s Selection
	s.Begin(Cell{Row: 0, Col: 0})
	s.Extend(Cell{Row: 0, Col: 40})
	if got := s.Text("hi        "); got != "hi" {
		t.Fatalf("Text = %q, want %q", got, "hi")
	}
}

func TestSelectionTextKeepsWideRunes(t *testing.T) {
	// The folder emoji is two columns wide, so the branch name starts at 10.
	view := "\U0001F4C1 mercury · dev"
	var s Selection
	s.Begin(Cell{Row: 0, Col: 3})
	s.Extend(Cell{Row: 0, Col: 10})
	if got := s.Text(view); got != "mercury" {
		t.Fatalf("Text = %q, want %q", got, "mercury")
	}
}

func TestSelectionPaintHighlightsOnlyTheRange(t *testing.T) {
	var s Selection
	s.Begin(Cell{Row: 0, Col: 6})
	s.Extend(Cell{Row: 0, Col: 11})
	painted := s.Paint(selectionView, highlight())

	if got := ansi.Strip(painted); got != selectionView {
		t.Fatalf("Paint changed the text: %q", got)
	}
	first := strings.Split(painted, "\n")[0]
	if !strings.Contains(first, "\x1b[") {
		t.Fatalf("Paint drew no escapes: %q", first)
	}
	for _, line := range strings.Split(painted, "\n")[1:] {
		if strings.Contains(line, "\x1b[") {
			t.Fatalf("Paint touched an unselected row: %q", line)
		}
	}
}

func TestSelectionPaintKeepsWidth(t *testing.T) {
	var s Selection
	s.Begin(Cell{Row: 0, Col: 2})
	s.Extend(Cell{Row: 2, Col: 4})
	for i, line := range strings.Split(s.Paint(selectionView, highlight()), "\n") {
		want := ansi.StringWidth(strings.Split(selectionView, "\n")[i])
		if got := ansi.StringWidth(line); got != want {
			t.Fatalf("row %d width = %d, want %d", i, got, want)
		}
	}
}

func TestSelectionPaintEmptyIsUntouched(t *testing.T) {
	var s Selection
	if got := s.Paint(selectionView, highlight()); got != selectionView {
		t.Fatalf("empty selection painted %q", got)
	}
}
