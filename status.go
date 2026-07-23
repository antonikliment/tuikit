package tuikit

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Status tracks a "press again to confirm" destructive-action flow together with
// the success/error message it leaves behind. One Status handles one pending
// action at a time; the target string disambiguates which item is armed.
type Status struct {
	pending string
	message string
	errText string
}

// Confirm implements press-again-to-confirm. On the first press for a target it
// arms that target (clearing any prior messages) and returns nil; on a second
// press of the same target it disarms and returns fire(). The confirm flag marks
// a dedicated confirm key (e.g. "y"): when true it will only fire an already
// armed target, never arm a new one.
func (s *Status) Confirm(target string, confirm bool, fire func() tea.Cmd) tea.Cmd {
	if s.pending != target {
		if confirm {
			return nil
		}
		s.pending, s.message, s.errText = target, "", ""
		return nil
	}
	s.pending = ""
	return fire()
}

// SetResult records the outcome of a fired action: on success it shows okMsg, on
// error it shows err.Error(). Either way it clears the pending target.
func (s *Status) SetResult(err error, okMsg string) {
	s.pending = ""
	s.message, s.errText = okMsg, ""
	if err != nil {
		s.message, s.errText = "", err.Error()
	}
}

// SetError shows msg as an error, superseding any success message. It leaves the
// armed target untouched.
func (s *Status) SetError(msg string) { s.errText, s.message = msg, "" }

// Disarm clears any armed target without touching the result message.
func (s *Status) Disarm() { s.pending = "" }

// Clear resets everything: armed target and any result message.
func (s *Status) Clear() { s.pending, s.message, s.errText = "", "", "" }

// Pending returns the currently armed target, or "" when nothing is armed.
func (s *Status) Pending() string { return s.pending }

// AppendRows appends a rendered status line to rows: a yellow error row if an
// error is set, otherwise a green success row if a message is set, otherwise
// rows unchanged.
func (s *Status) AppendRows(t Theme, rows []string) []string {
	if s.errText != "" {
		return append(rows, lipgloss.NewStyle().Foreground(t.Yellow).Render(s.errText))
	}
	if s.message != "" {
		return append(rows, lipgloss.NewStyle().Foreground(t.Green).Render(s.message))
	}
	return rows
}
