package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/antonikliment/tuikit"
)

// --- Table page: Table/Pairs alignment, StatusWord, and the formatters ---
//
// The point of the table below is that the STATE column is coloured and the
// SIZE column is not, yet both line up: widths are measured with display width,
// so a cell's escape sequences cost it no columns. Rendering the same table
// with len-based padding pushes every row after a coloured one to the right.

const (
	tickInterval  = 100 * time.Millisecond
	demoTotal     = int64(6_800_000_000)
	demoBytesTick = int64(140_000_000)
	tableGap      = 2
)

type tickMsg time.Time

type tableRow struct {
	name  string
	state string
	size  int64
	took  time.Duration
	age   time.Duration
}

var tableRows = []tableRow{
	{"gemma-2-2b-it", "running", 2_684_354_560, 272_914_939, 3*time.Hour + 23*time.Minute},
	{"llama-3.1-8b", "stopped", 8_589_934_592, 1_204_000, 47 * time.Minute},
	{"qwen2.5-coder-7b", "failed", 7_516_192_768, 89_412, 12 * time.Second},
	{"phi-4-mini", "ready", 402_653_184, 5_120_000_000, 26 * time.Hour},
}

type tablePage struct {
	theme    tuikit.Theme
	meter    tuikit.Meter
	received int64
	started  time.Time
	running  bool
}

func newTablePage() *tablePage {
	t := tuikit.DefaultTheme()
	return &tablePage{theme: t, meter: tuikit.NewMeter(24, t.Green)}
}

func (p *tablePage) Title() string { return "Table" }

func (p *tablePage) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyPressMsg:
		// The transfer is started by a keypress rather than on arrival: the Frame
		// consumes the digit that switched to this page, so there is no message
		// here to hang an initial tick on.
		if p.running {
			return nil
		}
		p.running, p.received, p.started = true, 0, time.Now()
		return tick()
	case tickMsg:
		if !p.running {
			return nil
		}
		if p.received += demoBytesTick; p.received >= demoTotal {
			p.received, p.running = demoTotal, false
			return nil
		}
		return tick()
	}
	return nil
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (p *tablePage) View(width, height int) string {
	t := p.theme
	rows := []string{t.StatusTitle("Table", "columns · formatters", t.Cyan, t.Green, width)}
	rows = append(rows, p.table()...)
	rows = append(rows, t.Rule(width), "")
	rows = append(rows, p.transfer()...)
	rows = append(rows, "", t.Rule(width), "")
	rows = append(rows, p.pairs()...)

	return t.PanelStyle(t.Cyan, false).Width(width).Height(max(3, height-2)).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// table renders the aligned table. Nothing here tracks widths by hand: Table
// measures the header alongside the body, so a column is never narrower than
// its own title, and the STATE column is coloured while the SIZE column is not
// yet both still line up.
//
// The state words are graded by StatusWord rather than by a switch this page
// owns — "running" and "ready" are green, "failed" red, anything unfamiliar
// amber, without the caller keeping its own list of what fine looks like.
func (p *tablePage) table() []string {
	t := p.theme
	now := time.Now()
	body := make([][]string, 0, len(tableRows))
	for _, r := range tableRows {
		body = append(body, []string{
			r.name,
			t.StatusWord(tuikit.Paint, r.state),
			tuikit.FormatBytes(r.size),
			tuikit.CoarseDuration(r.took).String(),
			tuikit.Age(now.Add(-r.age), now),
		})
	}
	block := t.Table(tuikit.Paint, []string{"name", "state", "size", "load", "seen"}, body, tableGap)
	return strings.Split(strings.TrimRight(block, "\n"), "\n")
}

// transfer shows TransferredBytes doing its two jobs: a known total renders as
// "x / y" beside a determinate meter, and an unknown one renders the received
// count alone, because a bar pinned at zero reads as a stall.
func (p *tablePage) transfer() []string {
	t := p.theme
	percent := int(float64(p.received) / float64(demoTotal) * 100)

	lines := []string{
		tuikit.Field("Download", fmt.Sprintf("%s %3d%%  %s",
			p.meter.View(percent), percent, tuikit.TransferredBytes(p.received, demoTotal))),
		tuikit.Field("No total", t.MutedStyle().Render(
			"downloading  "+tuikit.TransferredBytes(p.received, 0))),
	}
	switch {
	case p.running:
		lines = append(lines, tuikit.Field("Elapsed",
			tuikit.CoarseDuration(time.Since(p.started)).String()))
	case p.received > 0:
		lines = append(lines, tuikit.Field("Elapsed",
			t.Accent(t.Green).Render("complete in "+tuikit.CoarseDuration(time.Since(p.started)).String())))
	default:
		lines = append(lines, t.MutedStyle().Render("press any key to start the transfer"))
	}
	return lines
}

// pairs is the one-dimensional case: Pairs lines up a key column when the data
// is key/value rather than a table. Titleize turns the raw identifiers into
// labels, so "flash_attention" reads as "Flash attention".
func (p *tablePage) pairs() []string {
	t := p.theme
	keys := []string{"backend", "quantization", "context", "flash_attention"}
	values := []string{"llama.cpp", "Q4_K_M", "8192", t.StatusWord(tuikit.Paint, "enabled")}
	for i, key := range keys {
		keys[i] = tuikit.Titleize(key)
	}
	block := tuikit.IndentLines(t.Pairs(tuikit.Paint, keys, values, tableGap), 1)
	return append([]string{t.MutedStyle().Render("Engine arguments")},
		strings.Split(strings.TrimRight(block, "\n"), "\n")...)
}
