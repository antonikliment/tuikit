// Command syntax shows fenced code under different chroma stylesheets, switched
// while the program runs.
//
// The point is the switch, not the colors. A chroma stylesheet is baked into the
// renderer when [markdown.New] builds it, so changing it means building a new
// render function — and, because [tuikit.StreamingMarkdown] caches rendered
// output keyed on source and width alone, a new streamer too. Keep the old one
// and settled blocks stay in the previous palette forever, since their source
// never changes again.
//
//	go run ./examples/syntax
//
// Press tab to cycle stylesheets, or pick one with -syntax. The full set
// [markdown.SyntaxThemes] returns is what a host would offer in a picker.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/antonikliment/tuikit"
	"github.com/antonikliment/tuikit/markdown"
)

// Three stylesheets that differ enough to tell apart at a glance: the dark
// default, a light one, and a third with a distinctly different accent.
var themes = []string{markdown.DefaultSyntaxTheme, "tokyonight-day", "catppuccin-mocha"}

const sample = "# Syntax themes\n\n" +
	"Prose is colored from the `tuikit.Theme`; fenced code is colored by chroma.\n\n" +
	"```go\n" +
	`package main

import "fmt"

// greet builds the greeting line.
func greet(name string, n int) string {
	if n > 1 {
		return fmt.Sprintf("hello, %s x%d", name, n)
	}
	return "hello, " + name
}
` + "```\n\n" +
	"> The stylesheet changes; the prose around it does not.\n"

func main() {
	page := &syntaxPage{theme: tuikit.DefaultTheme()}
	page.use(0)

	frame := tuikit.New(
		tuikit.WithBrand("syntax", "chroma stylesheets, switched live"),
		tuikit.WithPages(page),
		tuikit.WithStatus(page.status),
	)
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type syntaxPage struct {
	theme    tuikit.Theme
	current  int
	streamer tuikit.StreamingMarkdown
}

func (p *syntaxPage) Title() string { return "syntax" }

// use swaps in the stylesheet at index i. Both the render function and the
// streamer are rebuilt: the first because New captures the stylesheet, the
// second because its cache is keyed on source and width, so it would happily
// serve the previous palette back.
func (p *syntaxPage) use(i int) {
	p.current = ((i % len(themes)) + len(themes)) % len(themes)
	render := markdown.New(p.theme, markdown.WithSyntaxTheme(themes[p.current]))
	p.streamer = tuikit.NewStreamingMarkdown(render)
}

func (p *syntaxPage) Update(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab", "right", "l":
			p.use(p.current + 1)
		case "shift+tab", "left", "h":
			p.use(p.current - 1)
		}
	}
	return nil
}

func (p *syntaxPage) View(width, height int) string {
	body := max(20, width-4)
	rendered := p.streamer.Render(sample, body)
	help := p.theme.SubtleStyle().Render("tab / shift+tab to change stylesheet")
	return p.theme.PanelStyle(p.theme.Brand, false).
		Width(width).
		Height(max(3, height-2)).
		Render(rendered + "\n\n" + help)
}

func (p *syntaxPage) status() (string, tuikit.Level) {
	name := themes[p.current]
	return fmt.Sprintf("%s  (%d of %d, %d available)",
		name, p.current+1, len(themes), len(markdown.SyntaxThemes())), tuikit.LevelInfo
}
