package tuikit

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}) }
func codeKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: c}) }

func typeQuery(s *SearchView, query string) {
	s.Update(runeKey('/')) // focus search
	for _, r := range query {
		s.Update(runeKey(r))
	}
}

func TestSearchViewFiltersBySubstring(t *testing.T) {
	s := NewSearchView()
	s.SetLines([]string{"alpha", "BETA", "gamma", "beta-two"})
	typeQuery(&s, "beta")
	got := s.Filtered()
	want := []string{"BETA", "beta-two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Filtered() = %v, want %v (case-insensitive)", got, want)
	}
}

func TestSearchViewEmptyQueryReturnsAllLines(t *testing.T) {
	s := NewSearchView()
	lines := []string{"one", "two", "three"}
	s.SetLines(lines)
	if got := s.Filtered(); !reflect.DeepEqual(got, lines) {
		t.Fatalf("Filtered() with no query = %v, want all %v", got, lines)
	}
}

func TestSearchViewSlashFocusesAndEscClears(t *testing.T) {
	s := NewSearchView()
	s.SetLines([]string{"needle", "haystack"})
	typeQuery(&s, "need")
	if !s.Searching() {
		t.Fatal("expected search input focused after '/'")
	}
	if s.Query() != "need" {
		t.Fatalf("Query() = %q, want %q", s.Query(), "need")
	}
	// Enter finishes typing but keeps the query.
	s.Update(codeKey(tea.KeyEnter))
	if s.Searching() {
		t.Fatal("Enter should blur the search input")
	}
	if s.Query() != "need" {
		t.Fatalf("query should survive Enter, got %q", s.Query())
	}
	// Esc (when not typing) clears the query.
	s.Update(codeKey(tea.KeyEscape))
	if s.Query() != "" {
		t.Fatalf("Esc should clear the query, got %q", s.Query())
	}
}

func TestSearchViewIgnoresNonKeyMessages(t *testing.T) {
	s := NewSearchView()
	s.SetLines([]string{"x"})
	s.Update(tea.WindowSizeMsg{Width: 10, Height: 5}) // must not panic or focus
	if s.Searching() {
		t.Fatal("a window-size message should not open search")
	}
}

func TestSearchViewViewRendersMatchingLines(t *testing.T) {
	s := NewSearchView()
	s.SetLines([]string{"apple", "banana", "cherry"})
	typeQuery(&s, "an") // matches "banana"
	out := s.View(40, 4)
	if !strings.Contains(out, "banana") {
		t.Fatalf("view should render matching line: %q", out)
	}
	if strings.Contains(out, "cherry") {
		t.Fatalf("view should not render non-matching line: %q", out)
	}
}
