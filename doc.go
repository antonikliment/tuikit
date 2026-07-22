// Package tuikit is a small, reusable Bubble Tea frame kit: the structural
// chrome you rebuild in every terminal app — a numbered page wrapper with
// navigation, chip tabs, and bordered panels — decoupled from any one app and
// driven by a swappable [Theme].
//
// # Frame and pages
//
// A [Frame] is a [tea.Model] that hosts a slice of pages, renders a numbered
// header ("[1] Foo  [2] Bar …"), delegates the body to the active page, and
// draws a status footer. Number keys 1-9 switch pages; Ctrl+C quits.
//
// A page is any value implementing [Page], a plain three-method interface:
//
//	type Page interface {
//		Title() string
//		Update(msg tea.Msg) tea.Cmd
//		View(width, height int) string
//	}
//
// Size is passed into View, so a page never tracks its own dimensions.
// Implement the page on a pointer type so Update can mutate state. A page may
// additionally implement [InputCapturer] so the Frame stops treating number
// keys as navigation while a field is focused.
//
// # Building blocks
//
// Pages assemble their bodies from the theme-driven helpers:
//
//   - [Theme.TabbedPanel] — a row of tabs joined seamlessly to a content panel;
//     the active tab opens into the panel with no dividing line.
//   - [Theme.TabStrip] — just the row of active/inactive tab chips.
//   - [Theme.PanelStyle] and [Panel] — bordered panels with a focused state.
//   - [Theme.StatusTitle], [Theme.Rule], [Field], [VerticalSlice], [Flow] —
//     layout helpers.
//
// See the docs/examples.md file for copy-paste snippets, and examples/demo for
// a runnable showcase.
package tuikit
