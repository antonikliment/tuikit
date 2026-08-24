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
//   - [SearchView] — a searchable viewport with follow-to-bottom behavior.
//   - [StreamingMarkdown] — markdown formatted as it arrives, settled blocks
//     rendered and the unfinished tail left raw. Pair it with a [RenderFunc];
//     [github.com/antonikliment/tuikit/markdown] supplies a glamour-backed one,
//     kept in its own package so the markdown stack stays optional.
//   - [Overlay] — a dismissable modal popup composited over a finished view,
//     pinnable to any of nine anchors via [Alignment].
//   - [Theme.ActionRow] — a labelled row of selectable actions.
//   - [Selection] and [Cell] — a mouse drag over a rendered frame, painted
//     with a highlight and copied out as plain text.
//   - [Help] and [HelpLine] — bright full and one-line keyboard help.
//   - [Meter] — a fixed-width resource gauge over bubbles/progress.
//   - [Status] — a press-again-to-confirm flow with a result message.
//   - [Theme.Table] and [Theme.Pairs] — an aligned block from a header and
//     rows, or from key/value pairs, over the display-width primitives
//     [Columns], [JoinCells], [Pad] and [Widest], so styled cells still align.
//   - [Painter], [PainterFor], [Paint], [Plain] — paint only when the
//     destination is a terminal and NO_COLOR is unset, decided once.
//   - [ClassifyStatus] and [Theme.StatusWord] — grade a status word into a
//     [Level] and paint it green, red or amber.
//   - [Theme.StatusTitle], [Theme.Rule], [Theme.EmptyPanel], [Field],
//     [VerticalSlice], [Flow], [AdaptiveWidth], [Indent], [IndentLines],
//     [TruncMiddle], [Titleize], [FormatBytes], [TransferredBytes],
//     [CoarseDuration], [Age] — layout and text helpers.
//
// See the docs/examples.md file for copy-paste snippets, and examples/demo for
// a runnable showcase.
package tuikit
