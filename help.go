package tuikit

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// Help returns a bubbles/help model with brighter key and description colors
// than the bubbles default, which renders very dim on many terminals. Use it
// directly for a full multi-column help layout, or call HelpLine for the common
// single-line short help.
func Help() help.Model {
	h := help.New()
	brightKey, brightDesc := lipgloss.Color("252"), lipgloss.Color("250")
	h.Styles.ShortKey = h.Styles.ShortKey.Foreground(brightKey)
	h.Styles.FullKey = h.Styles.FullKey.Foreground(brightKey)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Foreground(brightDesc)
	h.Styles.FullDesc = h.Styles.FullDesc.Foreground(brightDesc)
	return h
}

// defaultHelp backs HelpLine so callers get the brightened short help without
// holding their own help.Model.
var defaultHelp = Help()

// HelpLine renders bindings as a single "key desc • key desc" short-help line
// using the brightened styles.
func HelpLine(bindings ...key.Binding) string {
	return defaultHelp.ShortHelpView(bindings)
}
