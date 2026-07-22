package tuikit

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestNewDefaults(t *testing.T) {
	f := New()
	if f.ActivePage() != 0 {
		t.Fatalf("active = %d, want 0", f.ActivePage())
	}
	if !reflect.DeepEqual(f.Theme(), DefaultTheme()) {
		t.Fatal("New() should start from DefaultTheme")
	}
}

func TestWindowSizeMsgUpdatesSize(t *testing.T) {
	f := New()
	f, _ = updateFrame(f, tea.WindowSizeMsg{Width: 100, Height: 20})
	if f.width != 100 || f.height != 20 {
		t.Fatalf("size = %dx%d, want 100x20", f.width, f.height)
	}
}

func TestNumberKeySwitchesPageAndConsumesKey(t *testing.T) {
	a, b, c := &stubPage{title: "A"}, &stubPage{title: "B"}, &stubPage{title: "C"}
	f := New(WithPages(a, b, c))

	f, _ = updateFrame(f, keyMsg("2"))
	if f.ActivePage() != 1 {
		t.Fatalf("active = %d, want 1", f.ActivePage())
	}
	if b.updates != 0 {
		t.Fatalf("nav key should be consumed, page got %d updates", b.updates)
	}
}

func TestDigitBeyondRangeFallsThroughToPage(t *testing.T) {
	a, b := &stubPage{title: "A"}, &stubPage{title: "B"}
	f := New(WithPages(a, b))

	f, _ = updateFrame(f, keyMsg("3")) // only 2 pages
	if f.ActivePage() != 0 {
		t.Fatalf("active = %d, want 0 (unchanged)", f.ActivePage())
	}
	if a.updates != 1 || a.lastKey != "3" {
		t.Fatalf("out-of-range digit should reach active page: updates=%d key=%q", a.updates, a.lastKey)
	}
}

func TestNonNavKeyDelegatesToActivePage(t *testing.T) {
	a, b := &stubPage{title: "A"}, &stubPage{title: "B"}
	f := New(WithPages(a, b))

	updateFrame(f, keyMsg("x"))
	if a.updates != 1 || a.lastKey != "x" {
		t.Fatalf("active page updates=%d key=%q", a.updates, a.lastKey)
	}
	if b.updates != 0 {
		t.Fatalf("inactive page should not be updated, got %d", b.updates)
	}
}

func TestQuitReturnsQuitCommand(t *testing.T) {
	f := New(WithPages(&stubPage{title: "A"}))
	_, cmd := updateFrame(f, keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal("quit produced no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("command did not produce tea.QuitMsg, got %T", cmd())
	}
}

func TestInputCapturerSuppressesNavigation(t *testing.T) {
	a, b := &stubPage{title: "A", capturing: true}, &stubPage{title: "B"}
	f := New(WithPages(a, b))

	f, _ = updateFrame(f, keyMsg("2"))
	if f.ActivePage() != 0 {
		t.Fatalf("active = %d, want 0 (nav suppressed)", f.ActivePage())
	}
	if a.updates != 1 || a.lastKey != "2" {
		t.Fatalf("digit should reach capturing page: updates=%d key=%q", a.updates, a.lastKey)
	}
}

func TestGlobalKeysHandledBeforePage(t *testing.T) {
	a := &stubPage{title: "A"}
	handled := 0
	f := New(WithPages(a), WithGlobalKeys(func(msg tea.KeyPressMsg) (tea.Cmd, bool) {
		if msg.String() == "g" {
			handled++
			return nil, true
		}
		return nil, false
	}))

	f, _ = updateFrame(f, keyMsg("g"))
	if handled != 1 {
		t.Fatalf("global handler calls = %d, want 1", handled)
	}
	if a.updates != 0 {
		t.Fatalf("handled global key should not reach page, got %d updates", a.updates)
	}

	// Unhandled global keys fall through to the page.
	updateFrame(f, keyMsg("x"))
	if a.updates != 1 || a.lastKey != "x" {
		t.Fatalf("unhandled key should reach page: updates=%d key=%q", a.updates, a.lastKey)
	}
}

func TestGlobalKeysSkippedWhileCapturing(t *testing.T) {
	a := &stubPage{title: "A", capturing: true}
	called := false
	f := New(WithPages(a), WithGlobalKeys(func(tea.KeyPressMsg) (tea.Cmd, bool) {
		called = true
		return nil, true
	}))

	updateFrame(f, keyMsg("g"))
	if called {
		t.Fatal("global keys must be skipped while the page is capturing input")
	}
	if a.updates != 1 {
		t.Fatalf("key should reach capturing page, got %d updates", a.updates)
	}
}

func TestSetThemeMsgRethemesFrame(t *testing.T) {
	custom := DefaultTheme()
	custom.Brand = lipgloss.Color("205")

	f := New()
	f, _ = updateFrame(f, SetThemeMsg{Theme: custom})
	if !reflect.DeepEqual(f.Theme(), custom) {
		t.Fatal("SetThemeMsg did not re-theme the frame")
	}
}

func TestSetThemeCommand(t *testing.T) {
	custom := DefaultTheme()
	custom.Brand = lipgloss.Color("205")
	msg := SetTheme(custom)()
	if got, ok := msg.(SetThemeMsg); !ok || !reflect.DeepEqual(got.Theme, custom) {
		t.Fatalf("SetTheme() produced %#v", msg)
	}
}

func TestRenderShowsChrome(t *testing.T) {
	a, b := &stubPage{title: "Home", body: "HELLO-BODY"}, &stubPage{title: "Settings"}
	f := New(
		WithBrand("myapp", "tagline"),
		WithPages(a, b),
		WithStatus(func() (string, Level) { return "all good", LevelSuccess }),
	)
	f, _ = updateFrame(f, tea.WindowSizeMsg{Width: 120, Height: 24})

	out := ansi.Strip(f.render())
	for _, want := range []string{"myapp", "[1] Home", "[2] Settings", "HELLO-BODY", "all good"} {
		if !strings.Contains(out, want) {
			t.Fatalf("render missing %q:\n%s", want, out)
		}
	}
}

func TestRenderWithNoPagesDoesNotPanic(t *testing.T) {
	f := New(WithBrand("empty", ""))
	f, _ = updateFrame(f, tea.WindowSizeMsg{Width: 80, Height: 20})
	if out := ansi.Strip(f.render()); !strings.Contains(out, "empty") {
		t.Fatalf("render = %q", out)
	}
}
