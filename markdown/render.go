// Package markdown supplies a glamour-backed [tuikit.RenderFunc] for
// [tuikit.StreamingMarkdown], styled from a [tuikit.Theme].
//
// It is a separate package on purpose. A markdown engine is a large dependency
// — goldmark, chroma, an HTML sanitizer and a CSS parser — and the rest of
// tuikit needs none of it. Keeping it here means importing tuikit does not pull
// a markdown stack into a program that only wanted panels and tabs; only a
// program that imports this package links it.
package markdown

import (
	"fmt"
	"image/color"
	"strings"
	"sync"

	"github.com/antonikliment/tuikit"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"github.com/alecthomas/chroma/v2/styles"
)

// DefaultSyntaxTheme is the chroma stylesheet fenced code is highlighted with
// when no other is given. Chroma ships named stylesheets rather than taking
// colors, so a tuikit.Theme cannot drive highlighting the way it drives the
// rest of the styling — pick one that suits your palette with
// [WithSyntaxTheme]. The names come from chroma's styles package.
const DefaultSyntaxTheme = "tokyonight-night"

// SyntaxThemes lists the chroma stylesheet names [WithSyntaxTheme] accepts,
// sorted. It is here so a host can offer a picker without taking a chroma
// dependency of its own.
func SyntaxThemes() []string { return styles.Names() }

// Option configures the renderer.
type Option func(*config)

type config struct {
	syntax  string
	style   func(*ansi.StyleConfig)
	glamour []glamour.TermRendererOption
}

// WithSyntaxTheme sets the chroma stylesheet used for fenced code, overriding
// [DefaultSyntaxTheme]. An unknown name falls back to [DefaultSyntaxTheme]:
// chroma's own fallback is "swapoff", which colors six token types and leaves
// identifiers, operators and function names unstyled — a typo would render as
// almost-but-not-quite gray code rather than as an error. Light palettes want a
// light stylesheet — "tokyonight-day", "catppuccin-latte" and "github" are
// reasonable starts.
func WithSyntaxTheme(name string) Option {
	return func(c *config) { c.syntax = name }
}

// WithStyleFunc adjusts the stylesheet after the theme has been mapped onto it,
// for the elements a [tuikit.Theme] has no opinion about — a base text color, an
// inline-code background, a colored blockquote rule.
func WithStyleFunc(f func(*ansi.StyleConfig)) Option {
	return func(c *config) { c.style = f }
}

// WithGlamour passes options straight through to glamour, for anything this
// package does not surface.
func WithGlamour(opts ...glamour.TermRendererOption) Option {
	return func(c *config) { c.glamour = append(c.glamour, opts...) }
}

// New returns a render function that formats markdown with glamour, colored
// from theme, with fenced code highlighted by chroma. The returned function is
// safe to reuse across widths: it keeps one renderer per width, since building
// one parses a stylesheet and starts a goldmark pipeline — far too much to redo
// every frame.
//
// Rendering failures fall back to the input text: an answer that cannot be
// styled must still be readable.
//
// theme is captured here. A program whose theme changes at runtime needs a new
// render function, since the cached renderers are keyed on width alone.
func New(theme tuikit.Theme, opts ...Option) tuikit.RenderFunc {
	cfg := config{syntax: DefaultSyntaxTheme}
	for _, opt := range opts {
		opt(&cfg)
	}
	// chroma resolves an unknown stylesheet to its own near-monochrome
	// fallback, so an unrecognized name is corrected rather than passed on.
	if _, ok := styles.Registry[cfg.syntax]; !ok {
		cfg.syntax = DefaultSyntaxTheme
	}
	style := styleFor(theme, cfg.syntax)
	if cfg.style != nil {
		cfg.style(&style)
	}
	var mu sync.Mutex
	renderers := map[int]*glamour.TermRenderer{}

	return func(text string, width int) string {
		text = strings.TrimSpace(text)
		if text == "" || width <= 0 {
			return text
		}
		// glamour's Render mutates the goldmark parser's block stack, so a
		// renderer cannot be shared between concurrent calls; the map needs
		// guarding regardless.
		mu.Lock()
		defer mu.Unlock()

		renderer, ok := renderers[width]
		if !ok {
			options := append([]glamour.TermRendererOption{
				glamour.WithStyles(style),
				glamour.WithWordWrap(width),
			}, cfg.glamour...)
			built, err := glamour.NewTermRenderer(options...)
			if err != nil {
				return text
			}
			renderer, renderers[width] = built, built
		}
		out, err := renderer.Render(text)
		if err != nil {
			return text
		}
		return trimPadding(out)
	}
}

// trimPadding strips glamour's line padding and surrounding blank lines so the
// caller composes its own spacing rather than fighting the renderer's.
func trimPadding(out string) string {
	lines := strings.Split(strings.Trim(out, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

// styleFor maps a tuikit.Theme onto glamour's stylesheet. Only the elements the
// theme has an opinion about are set; everything else keeps glamour's default.
func styleFor(theme tuikit.Theme, syntax string) ansi.StyleConfig {
	muted, brand := hex(theme.Muted), hex(theme.Brand)
	blue, yellow := hex(theme.Blue), hex(theme.Yellow)
	border := hex(theme.FocusBorder)

	return ansi.StyleConfig{
		// No document margin: a host indents the block itself.
		Document:  block(ansi.StylePrimitive{}),
		Paragraph: block(ansi.StylePrimitive{}),
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: &muted, Italic: ptr(true)},
			Indent:         ptr(uint(1)),
			IndentToken:    ptr("│ "),
		},
		List:    ansi.StyleList{StyleBlock: block(ansi.StylePrimitive{}), LevelIndent: 2},
		Heading: block(ansi.StylePrimitive{Color: &yellow, Bold: ptr(true)}),
		H1:      block(ansi.StylePrimitive{Color: &yellow, Bold: ptr(true), Upper: ptr(true)}),
		H2:      block(ansi.StylePrimitive{Prefix: "", Color: &yellow, Bold: ptr(true)}),
		H3:      block(ansi.StylePrimitive{Prefix: "", Color: &yellow}),
		H4:      block(ansi.StylePrimitive{Prefix: "", Color: &yellow}),

		Text:           ansi.StylePrimitive{},
		Emph:           ansi.StylePrimitive{Italic: ptr(true)},
		Strong:         ansi.StylePrimitive{Bold: ptr(true)},
		Strikethrough:  ansi.StylePrimitive{CrossedOut: ptr(true)},
		HorizontalRule: ansi.StylePrimitive{Color: &border, Format: "\n──────\n"},

		Item:        ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{BlockPrefix: ". "},
		Task:        ansi.StyleTask{Ticked: "✓ ", Unticked: "· "},

		Link:     ansi.StylePrimitive{Color: &blue, Underline: ptr(true)},
		LinkText: ansi.StylePrimitive{Color: &blue},

		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: &brand}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{Margin: ptr(uint(2))},
			Theme:      syntax,
		},
		Table: ansi.StyleTable{
			StyleBlock:      block(ansi.StylePrimitive{}),
			CenterSeparator: ptr("┼"),
			ColumnSeparator: ptr("│"),
			RowSeparator:    ptr("─"),
		},
	}
}

// block is a style with no margin: the host indents the rendered answer itself,
// so glamour must not add its own.
func block(primitive ansi.StylePrimitive) ansi.StyleBlock {
	return ansi.StyleBlock{StylePrimitive: primitive, Margin: ptr(uint(0))}
}

func ptr[T any](v T) *T { return &v }

// hex renders a color as "#rrggbb". glamour's stylesheet takes colors as
// strings rather than as color.Color values.
func hex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}
