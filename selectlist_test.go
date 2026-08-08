package tuikit

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func modelList() SelectList {
	s := NewSelectList(DefaultTheme())
	s.SetItems([]SelectItem{
		{Label: "gemma-4-26B-A4B-it-MXFP4_MOE", Trailing: "local · 14.2 GB"},
		{Label: "gemma-4-E4B-it-GGUF", Trailing: "local · 4.3 GB"},
		{Label: "ornith-9b", Trailing: "active", Marked: true},
		{Label: "qwen-3-32b", Trailing: "local · 18.0 GB"},
	})
	return s
}

func typeFilter(s *SelectList, query string) {
	s.Update(runeKey('/'))
	for _, r := range query {
		s.Update(runeKey(r))
	}
}

func TestSelectListFiltersAndCountsMatches(t *testing.T) {
	s := modelList()
	if got := s.Counter(); got != "4" {
		t.Fatalf("Counter() unfiltered = %q, want %q", got, "4")
	}
	typeFilter(&s, "gemma")
	if got := len(s.Filtered()); got != 2 {
		t.Fatalf("Filtered() = %d rows, want 2", got)
	}
	if got := s.Counter(); got != "2 of 4" {
		t.Fatalf("Counter() = %q, want %q", got, "2 of 4")
	}
}

func TestSelectListFilterIsCaseInsensitive(t *testing.T) {
	s := modelList()
	typeFilter(&s, "GEMMA")
	if got := len(s.Filtered()); got != 2 {
		t.Fatalf("Filtered() = %d rows, want 2 (case-insensitive)", got)
	}
}

func TestSelectListCursorClampsWhenFilterNarrows(t *testing.T) {
	s := modelList()
	s.Update(codeKey(tea.KeyDown))
	s.Update(codeKey(tea.KeyDown))
	s.Update(codeKey(tea.KeyDown)) // cursor on the 4th row
	if got := s.SelectedKey(); got != "qwen-3-32b" {
		t.Fatalf("SelectedKey() = %q, want qwen-3-32b", got)
	}
	// Narrowing to two rows must pull the cursor back into range, not panic.
	typeFilter(&s, "gemma")
	if got := s.SelectedKey(); got != "gemma-4-E4B-it-GGUF" {
		t.Fatalf("SelectedKey() after narrowing = %q, want the last matching row", got)
	}
}

func TestSelectListCursorStopsAtEndsWithoutWrapping(t *testing.T) {
	s := modelList()
	s.Update(codeKey(tea.KeyUp)) // already at the top
	if got := s.SelectedKey(); got != "gemma-4-26B-A4B-it-MXFP4_MOE" {
		t.Fatalf("Up at the top should stay put, got %q", got)
	}
	for range 10 {
		s.Update(codeKey(tea.KeyDown))
	}
	if got := s.SelectedKey(); got != "qwen-3-32b" {
		t.Fatalf("Down past the end should stay on the last row, got %q", got)
	}
}

func TestSelectListNoMatchHasNoSelection(t *testing.T) {
	s := modelList()
	typeFilter(&s, "zzz")
	if _, ok := s.Selected(); ok {
		t.Fatal("Selected() should report false when nothing matches")
	}
	if got := s.SelectedKey(); got != "" {
		t.Fatalf("SelectedKey() = %q, want empty", got)
	}
	if out := s.View(40, 5); !strings.Contains(out, "no matches") {
		t.Fatalf("View() with no matches = %q", out)
	}
}

func TestSelectListKeyFallsBackToLabel(t *testing.T) {
	s := NewSelectList(DefaultTheme())
	s.SetItems([]SelectItem{{Label: "shown", Key: "resolved"}, {Label: "bare"}})
	if got := s.SelectedKey(); got != "resolved" {
		t.Fatalf("SelectedKey() = %q, want the explicit Key", got)
	}
	s.Update(codeKey(tea.KeyDown))
	if got := s.SelectedKey(); got != "bare" {
		t.Fatalf("SelectedKey() = %q, want the Label as fallback", got)
	}
}

func TestSelectListEscClosesFilterAndRestoresAllRows(t *testing.T) {
	s := modelList()
	typeFilter(&s, "gemma")
	if !s.Filtering() {
		t.Fatal("expected the filter focused after '/'")
	}
	s.Update(codeKey(tea.KeyEscape))
	if s.Filtering() {
		t.Fatal("Esc should blur the filter")
	}
	if got := len(s.Filtered()); got != 4 {
		t.Fatalf("Esc should clear the query, got %d rows", got)
	}
}

func TestSelectListEnterKeepsQueryAndBlurs(t *testing.T) {
	s := modelList()
	typeFilter(&s, "gemma")
	s.Update(codeKey(tea.KeyEnter))
	if s.Filtering() {
		t.Fatal("Enter should blur the filter")
	}
	if s.Query() != "gemma" {
		t.Fatalf("Query() = %q, want it to survive Enter", s.Query())
	}
}

func TestSelectListArrowsMoveWhileFiltering(t *testing.T) {
	s := modelList()
	typeFilter(&s, "gemma")
	s.Update(codeKey(tea.KeyDown))
	if !s.Filtering() {
		t.Fatal("arrows should not close the filter")
	}
	if got := s.SelectedKey(); got != "gemma-4-E4B-it-GGUF" {
		t.Fatalf("SelectedKey() = %q, want the second match", got)
	}
}

func TestSelectListViewScrollsToKeepCursorVisible(t *testing.T) {
	s := modelList()
	for range 3 {
		s.Update(codeKey(tea.KeyDown))
	}
	out := ansi.Strip(s.View(60, 2))
	if !strings.Contains(out, "qwen-3-32b") {
		t.Fatalf("View() should scroll the cursor into a 2-row window: %q", out)
	}
	if strings.Contains(out, "gemma-4-26B") {
		t.Fatalf("View() should have scrolled the first row out: %q", out)
	}
	if lines := strings.Split(out, "\n"); len(lines) != 2 {
		t.Fatalf("View(_, 2) rendered %d lines, want 2", len(lines))
	}
}

func TestSelectListViewTruncatesLabelFromTheMiddleKeepingTrailing(t *testing.T) {
	s := modelList()
	out := ansi.Strip(s.View(34, 4))
	first := strings.Split(out, "\n")[0]
	if !strings.Contains(first, "local · 14.2 GB") {
		t.Fatalf("trailing metadata should survive truncation: %q", first)
	}
	if !strings.Contains(first, "…") {
		t.Fatalf("long label should be truncated: %q", first)
	}
	// The distinctive tail is what middle-truncation exists to preserve.
	if !strings.Contains(first, "MOE") {
		t.Fatalf("middle truncation should keep the label tail: %q", first)
	}
	for _, line := range strings.Split(out, "\n") {
		if got := ansi.StringWidth(line); got > 34 {
			t.Fatalf("row width %d exceeds pane width 34: %q", got, line)
		}
	}
}

func TestSelectListIgnoresNonKeyMessages(t *testing.T) {
	s := modelList()
	s.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	if s.Filtering() {
		t.Fatal("a window-size message should not open the filter")
	}
}
