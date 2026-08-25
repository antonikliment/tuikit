// Command memolist is a runnable harness for [tuikit.MemoList]: a 5,000-message
// transcript in a 20-line viewport, with a live count of how many messages were
// actually rendered on the current frame.
//
//	go run ./examples/memolist
//	go run ./examples/memolist -items=50000
//
// Scroll with j/k or the arrows, page with u/d, jump with g/G, and q to quit.
// The number to watch is "rendered this frame": it stays in single digits
// however long the transcript is, and while the tail streams it is 1 — the
// message that changed — rather than the whole scrollback.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

func main() {
	var (
		count = flag.Int("items", 5000, "messages in the transcript")
		seed  = flag.Uint64("seed", 1, "random seed; fix it to replay a transcript exactly")
	)
	flag.Parse()

	theme := tuikit.DefaultTheme()
	m := &model{theme: theme, list: tuikit.NewMemoList()}
	rng := rand.New(rand.NewPCG(*seed, 0))
	items := make([]tuikit.ListItem, 0, *count+1)
	for i := range *count {
		items = append(items, newMessage(theme, i, rng, &m.renders))
	}
	m.tail = newMessage(theme, *count, rng, &m.renders)
	m.tail.body = "streaming"
	m.list.SetItems(append(items, m.tail))

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// tick grows the streaming tail by a word.
type tick time.Time

type model struct {
	theme tuikit.Theme
	list  tuikit.MemoList
	tail  *message

	// renders counts item renders. It is reset on every frame, so what the
	// footer shows is the cost of that frame alone.
	renders int
	frame   int
}

func (m *model) Init() tea.Cmd { return m.tick() }

func (m *model) tick() tea.Cmd {
	return tea.Tick(220*time.Millisecond, func(t time.Time) tea.Msg { return tick(t) })
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tick:
		// The streaming case: one message changes, and saying so costs the list
		// exactly that message's render on the next frame.
		m.tail.body += " " + words[m.frame%len(words)]
		m.frame++
		m.list.Invalidate(m.tail.ID())
		return m, m.tick()
	case tea.KeyPressMsg:
		return m, m.key(msg.String())
	}
	return m, nil
}

func (m *model) key(key string) tea.Cmd {
	switch key {
	case "q", "ctrl+c":
		return tea.Quit
	case "k", "up":
		m.list.ScrollBy(-1)
	case "j", "down":
		m.list.ScrollBy(1)
	case "u", "pgup":
		m.list.ScrollBy(-10)
	case "d", "pgdown":
		m.list.ScrollBy(10)
	case "g", "home":
		m.list.ScrollToTop()
	case "G", "end":
		m.list.ScrollToBottom()
	}
	return nil
}

const viewport = 20

func (m *model) View() tea.View {
	m.renders = 0
	body := m.list.Render(68, viewport)
	// Padding is the host's job: the list returns what it has, and a viewport
	// taller than the content is not its problem to fill.
	body += strings.Repeat("\n", max(viewport-lipgloss.Height(body), 0))

	muted := m.theme.Accent(m.theme.Muted)
	follow := m.theme.Accent(m.theme.Green).Render("following tail")
	if !m.list.Following() {
		follow = m.theme.Accent(m.theme.Yellow).Render("scrolled up")
	}
	status := fmt.Sprintf("%d messages · rendered this frame: %d · ", m.list.Len(), m.renders)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		m.theme.BrandStyle().Render("memolist"),
		"",
		m.theme.PanelStyle(m.theme.Brand, true).Width(74).Render(body),
		"",
		muted.Render(status)+follow,
		muted.Render("j/k scroll · u/d page · g/G top/bottom · q quit"),
	))
}

// message is one transcript entry: a speaker, a body, and a render counter it
// bumps whenever the list actually asks it to draw.
type message struct {
	theme   tuikit.Theme
	id      string
	who     string
	body    string
	renders *int
}

var words = strings.Fields(`the profile said the transcript was the cost so the list
now renders only what is on screen and remembers the rest which is why scrolling
a long session stopped getting slower the further back it went`)

func newMessage(theme tuikit.Theme, i int, rng *rand.Rand, renders *int) *message {
	who := "you"
	if i%2 == 1 {
		who = "agent"
	}
	n := 4 + rng.IntN(24)
	body := make([]string, n)
	for j := range body {
		body[j] = words[rng.IntN(len(words))]
	}
	return &message{
		theme:   theme,
		id:      fmt.Sprintf("m%d", i),
		who:     who,
		body:    fmt.Sprintf("#%d %s", i, strings.Join(body, " ")),
		renders: renders,
	}
}

func (m *message) ID() string { return m.id }

func (m *message) Render(width int) string {
	*m.renders++
	accent := m.theme.Cyan
	if m.who == "agent" {
		accent = m.theme.Green
	}
	head := m.theme.Accent(accent).Bold(true).Render(m.who)
	body := lipgloss.NewStyle().Width(max(width-2, 1)).Render(m.body)
	return head + "\n" + lipgloss.NewStyle().PaddingLeft(2).Render(body)
}
