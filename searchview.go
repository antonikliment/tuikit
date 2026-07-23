package tuikit

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// SearchView is a scrollable text pane with an incremental substring filter and
// follow-to-bottom behavior — the log/reader viewport most terminal apps
// rebuild by hand. Feed it lines with SetLines; it renders the subset matching
// the current query, stays pinned to the bottom as new lines arrive (until the
// user scrolls up), and toggles a search input on "/".
//
// SearchView owns no rendering chrome of its own beyond the viewport: a host
// composes the search prompt (see InputView) and any help footer around it, so
// it drops into an existing panel or tab layout.
type SearchView struct {
	vp     viewport.Model
	input  textinput.Model
	lines  []string
	follow bool
}

// NewSearchView returns a SearchView following the bottom of an empty pane.
func NewSearchView() SearchView {
	return SearchView{vp: viewport.New(), input: textinput.New(), follow: true}
}

// SetLines replaces the pane's backing content. The rendered view still applies
// the current query; call it every frame with fresh data (e.g. tailed logs).
func (s *SearchView) SetLines(lines []string) { s.lines = lines }

// Searching reports whether the search input has focus, so a host can stop
// treating typed keys as its own navigation while the user is typing a query.
func (s *SearchView) Searching() bool { return s.input.Focused() }

// Query is the current filter text.
func (s *SearchView) Query() string { return s.input.Value() }

// InputView renders the search input, for a host that wants to show the live
// "Search: …" prompt in its footer.
func (s *SearchView) InputView() string { return s.input.View() }

// Filtered returns the lines matching the current query (case-insensitive
// substring); the full slice when the query is empty. Matching is done against
// each line's visible text — ANSI styling is stripped first — so a query never
// matches the escape codes in a colored line, while the returned lines keep
// their styling for display.
func (s *SearchView) Filtered() []string {
	if s.input.Value() == "" {
		return s.lines
	}
	needle := strings.ToLower(s.input.Value())
	out := make([]string, 0, len(s.lines))
	for _, line := range s.lines {
		if strings.Contains(strings.ToLower(ansi.Strip(line)), needle) {
			out = append(out, line)
		}
	}
	return out
}

// Update handles a key message. While the search input is focused, keys type
// into it (Enter or Esc blur it). Otherwise "/" opens search, Esc clears the
// query and re-follows, and Up/Down scroll — scrolling away from the bottom
// stops follow, scrolling back to it resumes. Non-key messages are ignored.
func (s *SearchView) Update(msg tea.Msg) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return
	}
	if s.input.Focused() {
		switch key.String() {
		case "enter", "esc":
			s.input.Blur()
		default:
			s.input, _ = s.input.Update(key)
		}
		return
	}
	switch key.String() {
	case "/":
		s.input.Focus()
	case "esc":
		s.input.SetValue("")
		s.follow = true
	case "up":
		s.vp.ScrollUp(1)
		s.follow = s.vp.AtBottom()
	case "down":
		s.vp.ScrollDown(1)
		s.follow = s.vp.AtBottom()
	}
}

// View lays the filtered content into a width×height viewport, keeping the pane
// pinned to the bottom while following.
func (s *SearchView) View(width, height int) string {
	s.vp.SetWidth(max(1, width))
	s.vp.SetHeight(max(1, height))
	s.vp.SetContent(strings.Join(s.Filtered(), "\n"))
	if s.follow {
		s.vp.GotoBottom()
	}
	return s.vp.View()
}
