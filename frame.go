package tuikit

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Page is a single screen hosted by a Frame. Implement it on a pointer type so
// Update can mutate state. Size is passed into View, so a page never tracks its
// own width/height.
type Page interface {
	// Title is the tab label shown in the numbered header.
	Title() string
	// Update handles a message addressed to this page (it is the active page).
	Update(msg tea.Msg) tea.Cmd
	// View renders the page body into the given content area.
	View(width, height int) string
}

// InputCapturer is an optional Page capability: while CapturingInput reports
// true (e.g. a focused search field), the Frame stops treating number keys as
// page navigation and forwards them to the page instead.
type InputCapturer interface {
	CapturingInput() bool
}

// Level classifies footer status text.
type Level int

const (
	LevelInfo Level = iota
	LevelSuccess
	LevelWarning
)

// StatusFunc supplies the footer status line. Return an empty string to show
// the default "Ready".
type StatusFunc func() (text string, level Level)

// Frame is the stateful page wrapper: a numbered header, a body delegated to
// the active Page, and a status footer. It implements tea.Model, so it can be
// handed straight to tea.NewProgram.
type Frame struct {
	theme   Theme
	brand   string
	tagline string
	pages   []Page
	active  int
	keys    KeyMap
	status  StatusFunc

	width  int
	height int
}

// Option configures a Frame.
type Option func(*Frame)

// WithTheme overrides the default palette.
func WithTheme(t Theme) Option { return func(f *Frame) { f.theme = t } }

// WithBrand sets the header brand name and an optional tagline beside it.
func WithBrand(brand, tagline string) Option {
	return func(f *Frame) { f.brand, f.tagline = brand, tagline }
}

// WithPages sets the pages, in tab order.
func WithPages(pages ...Page) Option { return func(f *Frame) { f.pages = pages } }

// WithStatus sets the footer status provider.
func WithStatus(status StatusFunc) Option { return func(f *Frame) { f.status = status } }

// WithKeyMap overrides the navigation bindings.
func WithKeyMap(k KeyMap) Option { return func(f *Frame) { f.keys = k } }

// New builds a Frame. Defaults: DefaultTheme, DefaultKeyMap, 120x32 until the
// first WindowSizeMsg.
func New(opts ...Option) *Frame {
	f := &Frame{theme: DefaultTheme(), keys: DefaultKeyMap(), width: 120, height: 32}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Theme returns the frame's theme, handy for pages that want to match it.
func (f *Frame) Theme() Theme { return f.theme }

// ActivePage returns the current page index.
func (f *Frame) ActivePage() int { return f.active }

// Init implements tea.Model.
func (f *Frame) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (f *Frame) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width, f.height = msg.Width, msg.Height
		return f, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, f.keys.Quit) {
			return f, tea.Quit
		}
		if !f.activeCapturingInput() {
			if idx, ok := pageDigit(msg.String()); ok && idx < len(f.pages) {
				f.active = idx
				return f, nil
			}
		}
	}
	if len(f.pages) == 0 {
		return f, nil
	}
	return f, f.pages[f.active].Update(msg)
}

func (f *Frame) activeCapturingInput() bool {
	if len(f.pages) == 0 {
		return false
	}
	if c, ok := f.pages[f.active].(InputCapturer); ok {
		return c.CapturingInput()
	}
	return false
}

// View implements tea.Model.
func (f *Frame) View() tea.View {
	view := tea.NewView(f.render())
	view.AltScreen = true
	if f.brand != "" {
		view.WindowTitle = f.brand
	}
	return view
}

func (f *Frame) render() string {
	width := max(f.width, 20)
	inner := max(width-4, 16)

	body := ""
	if len(f.pages) > 0 {
		body = f.pages[f.active].View(inner, max(1, f.height-4))
	}

	app := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(f.theme.Muted).Padding(0, 1)
	return app.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, f.header(inner), body, f.footer(inner)),
	)
}

func (f *Frame) header(width int) string {
	brand := f.theme.BrandStyle().Render(f.brand)
	if f.tagline != "" {
		brand += f.theme.MutedStyle().Render("  " + f.tagline)
	}
	labels := make([]string, 0, len(f.pages)*2)
	for i, page := range f.pages {
		if i > 0 {
			labels = append(labels, "    ")
		}
		labels = append(labels, f.tabLabel(i, page.Title()))
	}
	nav := lipgloss.JoinHorizontal(lipgloss.Top, labels...)
	quit := f.theme.MutedStyle().Render("Ctrl+C Quit")
	mid := max(1, width-lipgloss.Width(brand)-lipgloss.Width(quit))
	line := brand + lipgloss.PlaceHorizontal(mid, lipgloss.Center, nav) + quit
	return lipgloss.NewStyle().Width(width).Render(line)
}

func (f *Frame) tabLabel(index int, title string) string {
	text := fmt.Sprintf("[%d] %s", index+1, title)
	if index == f.active {
		return lipgloss.NewStyle().Foreground(f.theme.Blue).Bold(true).Underline(true).Render(text)
	}
	return f.theme.MutedStyle().Render(text)
}

func (f *Frame) footer(width int) string {
	text, level := "Ready", LevelInfo
	if f.status != nil {
		if t, l := f.status(); t != "" {
			text, level = t, l
		}
	}
	style := f.theme.SubtleStyle()
	switch level {
	case LevelWarning:
		style = f.theme.Accent(f.theme.Yellow)
	case LevelSuccess:
		style = f.theme.Accent(f.theme.Green)
	}
	return lipgloss.NewStyle().Width(width).Render(style.Render("Status: " + text))
}
