# tuikit examples

Copy-paste snippets for each piece of the kit. A full runnable program lives in
[`examples/demo`](../examples/demo); run it with `go run ./examples/demo`.

- [Minimal app](#minimal-app)
- [Writing a page](#writing-a-page)
- [Sub-tabs with TabStrip](#sub-tabs-with-tabstrip)
- [Panels](#panels)
- [A scrolling page (viewport)](#a-scrolling-page-viewport)
- [Search that suppresses page nav (InputCapturer)](#search-that-suppresses-page-nav-inputcapturer)
- [Footer status](#footer-status)
- [Custom theme](#custom-theme)
- [Layout helpers](#layout-helpers)

## Minimal app

```go
package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/antonikliment/tuikit"
)

func main() {
	frame := tuikit.New(
		tuikit.WithBrand("myapp", "does a thing"),
		tuikit.WithPages(newHomePage(), newSettingsPage()),
	)
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		panic(err)
	}
}
```

`Frame` implements `tea.Model`, so it goes straight into `tea.NewProgram`. It
sets alt-screen and the window title itself.

## Writing a page

A page is a plain three-method interface. Implement it on a pointer so `Update`
can mutate state; `View` receives the content area size, so you never track it
yourself.

```go
type homePage struct {
	theme tuikit.Theme
	hits  int
}

func newHomePage() *homePage { return &homePage{theme: tuikit.DefaultTheme()} }

func (p *homePage) Title() string { return "Home" }

func (p *homePage) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" {
		p.hits++
	}
	return nil
}

func (p *homePage) View(width, height int) string {
	return p.theme.PanelStyle(p.theme.Cyan, false).
		Width(width).Height(height - 2).
		Render(fmt.Sprintf("Enter pressed %d times", p.hits))
}
```

## Sub-tabs with TabStrip

`TabStrip` renders a chip row — the active chip is filled with its accent, the
rest are muted. Titles are pre-formatted, so you can append counts.

```go
func (p *page) View(width, height int) string {
	titles  := []string{"Presets (3)", "Local models (7)"}
	accents := []color.Color{p.theme.Cyan, p.theme.Green}
	strip   := p.theme.TabStrip(titles, accents, p.focus) // p.focus is 0 or 1
	// ... render the active table/body below the strip ...
	return lipgloss.JoinVertical(lipgloss.Left, strip, body)
}

func (p *page) Update(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		n := 2
		switch k.String() {
		case "tab":       p.focus = (p.focus + 1) % n
		case "shift+tab": p.focus = (p.focus + n - 1) % n
		}
	}
	return nil
}
```

Tab/Shift+Tab are free for page-internal use — the Frame only navigates on the
number keys.

## Panels

```go
// Style form — compose with your own width/height:
s := theme.PanelStyle(theme.Cyan, focused) // focused => double border
out := s.Width(w).Height(h).Render(content)

// Struct form — size baked in (zero W/H means "fit content"):
out = tuikit.Panel{Theme: theme, Accent: theme.Green, Focused: true, Width: w, Height: h}.Render(content)
```

## A scrolling page (viewport)

Hold a `bubbles/v2/viewport`, forward messages to it in `Update`, and set its
size from `View`.

```go
type readerPage struct {
	theme tuikit.Theme
	vp    viewport.Model
}

func newReaderPage(text string) *readerPage {
	vp := viewport.New()
	vp.SetContent(text)
	return &readerPage{theme: tuikit.DefaultTheme(), vp: vp}
}

func (p *readerPage) Title() string { return "Reader" }

func (p *readerPage) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.vp, cmd = p.vp.Update(msg) // handles ↑/↓, PgUp/PgDn
	return cmd
}

func (p *readerPage) View(width, height int) string {
	p.vp.SetWidth(width - 4)   // panel border + padding
	p.vp.SetHeight(height - 4) // strip + panel border
	strip := p.theme.Accent(p.theme.Cyan).
		Render(fmt.Sprintf("scroll: %3.0f%%", p.vp.ScrollPercent()*100))
	panel := p.theme.PanelStyle(p.theme.Cyan, false).
		Width(width).Height(height - 2).Render(p.vp.View())
	return lipgloss.JoinVertical(lipgloss.Left, strip, panel)
}
```

## Search that suppresses page nav (InputCapturer)

Implement `InputCapturer` so that while your input is focused, the Frame hands
number keys to the page (to type) instead of switching pages.

```go
type searchPage struct {
	theme tuikit.Theme
	input textinput.Model
	items []string
}

// CapturingInput => the Frame stops treating 1-9 as navigation.
func (p *searchPage) CapturingInput() bool { return p.input.Focused() }

func (p *searchPage) Update(msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if p.input.Focused() {
		if s := k.String(); s == "enter" || s == "esc" {
			p.input.Blur()
			return nil
		}
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return cmd
	}
	if k.String() == "/" {
		return p.input.Focus()
	}
	return nil
}

func (p *searchPage) filtered() []string {
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if q == "" {
		return p.items
	}
	out := p.items[:0:0]
	for _, it := range p.items {
		if strings.Contains(strings.ToLower(it), q) {
			out = append(out, it)
		}
	}
	return out
}
```

## Footer status

Provide a `StatusFunc`; return a `Level` to color the line (info/success/warning).

```go
tuikit.WithStatus(func() (string, tuikit.Level) {
	if err != nil {
		return err.Error(), tuikit.LevelWarning
	}
	return "Ready", tuikit.LevelInfo
})
```

## Custom theme

Everything draws from a `Theme`; swap it with `WithTheme`. Start from the
default and override, or build one field-by-field.

```go
theme := tuikit.DefaultTheme()
theme.Brand = lipgloss.Color("205") // pink brand
theme.Cyan  = lipgloss.Color("39")

frame := tuikit.New(
	tuikit.WithTheme(theme),
	tuikit.WithPages(/* ... */),
)
```

## Layout helpers

```go
t := tuikit.DefaultTheme()

t.StatusTitle("chat", "Running", t.Cyan, t.Green, width) // "chat ....... ● Running"
tuikit.Field("Model", "gemma.gguf")                      // "Model:    gemma.gguf"
t.Rule(width)                                            // muted divider line
tuikit.VerticalSlice(content, 0, height)                 // hard-clip to height lines
tuikit.Flow(width, 2, blocks)                            // wrap blocks left-to-right
```
