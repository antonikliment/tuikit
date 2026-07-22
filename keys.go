package tuikit

import "charm.land/bubbles/v2/key"

// KeyMap holds the Frame-level navigation bindings. Page switching is by number
// key so that Tab/Shift+Tab stay free for pages to use internally (e.g. panel
// focus). Override via WithKeyMap.
type KeyMap struct {
	// PageDigits documents the number-key navigation for help output; the Frame
	// matches digits 1-9 directly regardless of this binding's keys.
	PageDigits key.Binding
	Quit       key.Binding
}

// DefaultKeyMap returns the stock bindings: 1-9 switch pages, Ctrl+C quits.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		PageDigits: key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"), key.WithHelp("1-9", "Switch page")),
		Quit:       key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("Ctrl+C", "Quit")),
	}
}

// pageDigit reports the 0-based page index for a "1".."9" keypress.
func pageDigit(s string) (int, bool) {
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		return int(s[0] - '1'), true
	}
	return 0, false
}
