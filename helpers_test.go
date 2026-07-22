package tuikit

import (
	tea "charm.land/bubbletea/v2"
)

// keyMsg builds a KeyPressMsg whose String() is value (e.g. "tab", "2",
// "ctrl+c"), matching how bubbletea reports keys.
func keyMsg(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: value, Code: []rune(value)[0]})
}

// stubPage is a test-double Page that records the messages it receives and can
// pretend to capture input.
type stubPage struct {
	title     string
	body      string
	capturing bool
	updates   int
	lastKey   string
}

func (p *stubPage) Title() string { return p.title }

func (p *stubPage) Update(msg tea.Msg) tea.Cmd {
	p.updates++
	if k, ok := msg.(tea.KeyPressMsg); ok {
		p.lastKey = k.String()
	}
	return nil
}

func (p *stubPage) View(width, height int) string { return p.body }

func (p *stubPage) CapturingInput() bool { return p.capturing }

// updateFrame applies a message and returns the frame back as *Frame.
func updateFrame(f *Frame, msg tea.Msg) (*Frame, tea.Cmd) {
	model, cmd := f.Update(msg)
	return model.(*Frame), cmd
}
