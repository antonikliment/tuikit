package tuikit

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SelectItem is one row of a [SelectList]: a label the user picks by, plus
// optional right-aligned metadata. Key is what the host resolves the selection
// back to; when empty, Label stands in.
type SelectItem struct {
	// Label is the primary text. It is filtered against and middle-truncated
	// when the pane is too narrow, so the distinctive tail of a long name
	// (a quantization suffix, a path leaf) survives.
	Label string
	// Trailing is right-aligned metadata — "local · 14.2 GB", "2h ago".
	Trailing string
	// Marked draws Trailing in the theme's Yellow instead of muted, for the
	// one row that is already current ("active").
	Marked bool
	// Key is the host's identifier for this row. Falls back to Label.
	Key string
}

func (i SelectItem) key() string {
	if i.Key != "" {
		return i.Key
	}
	return i.Label
}

// SelectList is a filterable, cursor-driven picker — the "choose one of these,
// and let me type to narrow it down" list most terminal apps rebuild by hand
// around bubbles/list and a custom delegate.
//
// It complements [SearchView], which filters a scrolling text pane but has no
// selection. Like SearchView it owns no chrome beyond its rows: a host composes
// the surrounding panel, footer, and counter (see [SelectList.Counter]).
type SelectList struct {
	theme  Theme
	input  textinput.Model
	items  []SelectItem
	cursor int
	offset int
}

// NewSelectList returns an empty SelectList drawing from theme, with the filter
// closed.
func NewSelectList(theme Theme) SelectList {
	in := textinput.New()
	in.Prompt = ""
	return SelectList{theme: theme, input: in}
}

// SetItems replaces the backing rows, clamping the cursor into the new filtered
// set. Call it whenever the source data changes.
func (s *SelectList) SetItems(items []SelectItem) {
	s.items = items
	s.clamp()
}

// SetTheme reskins the list, for a host that rebuilds its palette when the
// terminal background changes.
func (s *SelectList) SetTheme(theme Theme) { s.theme = theme }

// Focus opens the filter and puts the cursor in it, for a host that wants the
// list to be type-to-filter from the moment it opens.
func (s *SelectList) Focus() { s.input.Focus() }

// Filtering reports whether the filter input has focus, so a host can stop
// treating typed keys as its own shortcuts while the user narrows the list.
func (s *SelectList) Filtering() bool { return s.input.Focused() }

// Query is the current filter text.
func (s *SelectList) Query() string { return s.input.Value() }

// InputView renders the filter input, for a host showing a live prompt.
func (s *SelectList) InputView() string { return s.input.View() }

// Filtered returns the rows matching the current query (case-insensitive
// substring over Label), or every row when the query is empty.
func (s *SelectList) Filtered() []SelectItem {
	if s.input.Value() == "" {
		return s.items
	}
	needle := strings.ToLower(s.input.Value())
	out := make([]SelectItem, 0, len(s.items))
	for _, item := range s.items {
		if strings.Contains(strings.ToLower(item.Label), needle) {
			out = append(out, item)
		}
	}
	return out
}

// Selected is the row under the cursor. The bool is false when the filter
// matches nothing.
func (s *SelectList) Selected() (SelectItem, bool) {
	rows := s.Filtered()
	if s.cursor < 0 || s.cursor >= len(rows) {
		return SelectItem{}, false
	}
	return rows[s.cursor], true
}

// SelectedKey is [SelectList.Selected]'s Key, or "" when nothing matches.
func (s *SelectList) SelectedKey() string {
	item, ok := s.Selected()
	if !ok {
		return ""
	}
	return item.key()
}

// Counter is the "9 of 21" progress label, or "21" when nothing is filtered
// out — there is no news in "21 of 21".
func (s *SelectList) Counter() string {
	shown, total := len(s.Filtered()), len(s.items)
	if shown == total {
		return fmt.Sprintf("%d", total)
	}
	return fmt.Sprintf("%d of %d", shown, total)
}

// Update handles a key message. While the filter is focused, printable keys
// type into it and Enter or Esc closes it (Esc also clearing the query).
// Otherwise "/" opens the filter and Up/Down move the cursor. Enter is left for
// the host to act on — SelectList never decides what picking means. Non-key
// messages are ignored.
func (s *SelectList) Update(msg tea.Msg) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return
	}
	if s.input.Focused() {
		s.updateFiltering(key)
		return
	}
	switch key.String() {
	case "/":
		s.input.Focus()
	case "up":
		s.move(-1)
	case "down":
		s.move(1)
	}
}

// updateFiltering routes a key while the filter has focus. Arrows still move the
// cursor so the user can narrow and pick without leaving the input.
func (s *SelectList) updateFiltering(key tea.KeyPressMsg) {
	switch key.String() {
	case "esc":
		s.input.SetValue("")
		s.input.Blur()
		s.clamp()
	case "enter":
		s.input.Blur()
	case "up":
		s.move(-1)
	case "down":
		s.move(1)
	default:
		s.input, _ = s.input.Update(key)
		s.clamp()
	}
}

// move steps the cursor by delta, stopping at either end rather than wrapping.
func (s *SelectList) move(delta int) {
	s.cursor = min(max(s.cursor+delta, 0), max(len(s.Filtered())-1, 0))
}

// clamp keeps the cursor inside the filtered set after the query or items
// change.
func (s *SelectList) clamp() {
	s.cursor = min(max(s.cursor, 0), max(len(s.Filtered())-1, 0))
}

// View renders up to height rows in width columns, scrolling a window that
// keeps the cursor visible. The selected row is marked with "›" and drawn in
// the theme's Brand color; labels too long for the pane are middle-truncated.
func (s *SelectList) View(width, height int) string {
	rows := s.Filtered()
	if len(rows) == 0 {
		return s.theme.MutedStyle().Render("no matches")
	}
	height = max(height, 1)
	s.scroll(len(rows), height)

	lines := make([]string, 0, height)
	for i := s.offset; i < len(rows) && i < s.offset+height; i++ {
		lines = append(lines, s.row(rows[i], i == s.cursor, max(width, 1)))
	}
	return strings.Join(lines, "\n")
}

// scroll slides the visible window so the cursor stays inside it.
func (s *SelectList) scroll(total, height int) {
	s.offset = min(s.offset, max(total-height, 0))
	s.offset = min(s.offset, s.cursor)
	if s.cursor >= s.offset+height {
		s.offset = s.cursor - height + 1
	}
	s.offset = max(s.offset, 0)
}

// row draws one line: "› label      trailing", with the label truncated from
// the middle to leave the trailing metadata intact.
func (s *SelectList) row(item SelectItem, selected bool, width int) string {
	trailing := item.Trailing
	trailWidth := 0
	if trailing != "" {
		trailWidth = lipgloss.Width(trailing) + 2
	}

	label := TruncMiddle(item.Label, max(width-2-trailWidth, 1))
	marker, labelStyle := "  ", lipgloss.NewStyle().Foreground(s.theme.Muted)
	if selected {
		marker = s.theme.Accent(s.theme.Brand).Render("› ")
		labelStyle = s.theme.Accent(s.theme.Brand).Bold(true)
	}

	line := marker + labelStyle.Render(label)
	if trailing == "" {
		return line
	}
	trailStyle := s.theme.SubtleStyle()
	if item.Marked {
		trailStyle = s.theme.Accent(s.theme.Yellow)
	}
	gap := max(width-lipgloss.Width(line)-lipgloss.Width(trailing), 1)
	return line + strings.Repeat(" ", gap) + trailStyle.Render(trailing)
}
