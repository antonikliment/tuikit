// Command streaming is a runnable harness for [tuikit.StreamingMarkdown]: an
// endless generated markdown document, streamed a character at a time, so the
// boundary handling can be watched rather than inferred from tests.
//
// Watch for a code fence appearing as plain text and snapping to highlighted
// when it closes, the boundary advancing steadily rather than stalling, and no
// corrupted output at high rates.
//
//	go run ./examples/streaming -seed=7 -cps=80 -block=3 -debug
//	go run ./examples/streaming -seed=7 -cps=400 -block=12   # long fences
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/antonikliment/tuikit"
	"github.com/antonikliment/tuikit/markdown"
)

func main() {
	var (
		seed   = flag.Uint64("seed", 1, "random seed; fix it to replay a stream exactly")
		cps    = flag.Int("cps", 80, "characters per second")
		block  = flag.Int("block", 3, "block size: sentences per paragraph, items per list, rows per table")
		debug  = flag.Bool("debug", false, "mark the boundary and report prefix/tail sizes")
		syntax = flag.String("syntax", markdown.DefaultSyntaxTheme, "chroma stylesheet for fenced code")
	)
	flag.Parse()

	theme := tuikit.DefaultTheme()
	render := markdown.New(theme, markdown.WithSyntaxTheme(*syntax))
	page := &streamPage{
		theme:     theme,
		render:    render,
		streamer:  tuikit.NewStreamingMarkdown(render),
		generator: newGenerator(*seed, *block),
		view:      tuikit.NewSearchView(),
		debug:     *debug,
		interval:  time.Second / time.Duration(max(1, *cps)),
	}

	frame := tuikit.New(
		tuikit.WithBrand("streaming", "markdown as it arrives"),
		tuikit.WithPages(page),
		tuikit.WithStatus(page.status),
	)
	if _, err := tea.NewProgram(&ticking{Model: frame, start: page.schedule}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ticking starts the page's clock. A Frame's Init returns no command and
// window messages are not forwarded to pages, so a page that drives itself has
// no way to schedule its first tick — this wrapper hands it one and is
// otherwise transparent. Worth revisiting in the kit if more self-driving pages
// appear.
type ticking struct {
	tea.Model
	start func() tea.Cmd
}

func (t *ticking) Init() tea.Cmd { return t.start() }

// tick advances the stream by one character.
type tick time.Time

type streamPage struct {
	theme     tuikit.Theme
	streamer  tuikit.StreamingMarkdown
	render    tuikit.RenderFunc
	generator *generator
	view      tuikit.SearchView
	interval  time.Duration
	debug     bool
	paused    bool

	buffer   string // everything emitted so far
	pending  string // the block being fed in, character by character
	rendered string
}

func (p *streamPage) Title() string { return "stream" }

func (p *streamPage) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case " ":
			p.paused = !p.paused
		case "c":
			p.buffer, p.pending = "", ""
		case "d":
			p.debug = !p.debug
		}
	case tick:
		if !p.paused {
			p.advance()
		}
		return p.schedule()
	}
	p.view.Update(msg)
	return nil
}

func (p *streamPage) schedule() tea.Cmd {
	return tea.Tick(p.interval, func(t time.Time) tea.Msg { return tick(t) })
}

// advance moves one character from the pending block into the buffer, pulling a
// fresh block when the current one runs out. Feeding a character at a time is
// the worst case for boundary detection, which is the point.
func (p *streamPage) advance() {
	if p.pending == "" {
		p.pending = p.generator.next()
	}
	p.buffer += p.pending[:1]
	p.pending = p.pending[1:]
}

func (p *streamPage) View(width, height int) string {
	body := max(1, width-4)
	p.rendered = p.streamer.Render(p.buffer, body)

	content := p.rendered
	if p.debug {
		content = p.annotate(body)
	}
	p.view.SetLines(strings.Split(content, "\n"))
	return p.theme.PanelStyle(p.theme.Brand, false).
		Width(width).
		Height(max(3, height-2)).
		Render(p.view.View(body, max(1, height-4)))
}

// annotate rebuilds the view with a rule at the cut, so the boundary is visible
// as it advances. It renders the two halves itself rather than splicing the
// component's output: the settled part and the tail are joined by a blank line,
// but so are blocks *within* each half, so there is no offset to splice at.
func (p *streamPage) annotate(width int) string {
	settled, tail := p.split()
	rule := p.theme.SubtleStyle().Render(strings.Repeat("─", max(4, width/2)) + " boundary")

	parts := make([]string, 0, 3)
	if settled != "" {
		parts = append(parts, p.render(settled, width))
	}
	parts = append(parts, rule)
	if trimmed := strings.TrimSpace(tail); trimmed != "" {
		parts = append(parts, ansi.Wrap(trimmed, width, ""))
	}
	return strings.Join(parts, "\n")
}

// split reports how the buffer is currently divided, asking the renderer rather
// than re-deriving it: a marker that guessed the boundary would be lying in the
// one place the boundary is meant to be visible.
func (p *streamPage) split() (settled, tail string) {
	cut := min(p.streamer.Settled(), len(p.buffer))
	return p.buffer[:cut], p.buffer[cut:]
}

func (p *streamPage) status() (string, tuikit.Level) {
	settled, tail := p.split()
	state := "streaming"
	level := tuikit.LevelSuccess
	if p.paused {
		state, level = "paused", tuikit.LevelWarning
	}
	if !p.debug {
		return state + " — space pauses, c clears, d shows the boundary", level
	}
	fenced := strings.Count(settled+tail, "```")%2 == 1
	return fmt.Sprintf("%s — prefix %dB | tail %dB | fenced %v", state, len(settled), len(tail), fenced), level
}
