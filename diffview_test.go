package tuikit

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// update rewrites the golden files instead of comparing against them:
//
//	go test -run Golden -update ./...
var update = flag.Bool("update", false, "rewrite golden files")

const diffBefore = `package main

import "fmt"

func greet(name string) string {
	return "hello, " + name
}

func main() {
	fmt.Println(greet("world"))
}
`

const diffAfter = `package main

import "fmt"

func greet(name string, times int) string {
	if times > 1 {
		return fmt.Sprintf("hello, %s x%d", name, times)
	}
	return "hi, " + name
}

func main() {
	fmt.Println(greet("world", 2))
}
`

// plainDiff is the builder the golden tests share: no escapes and no
// highlighting, so a golden file is readable and a chroma release cannot
// rewrite every one of them.
func plainDiff() *DiffView {
	return NewDiffView(DefaultTheme()).
		Painter(Plain).
		Highlighter(nil).
		Before("greet.go", diffBefore).
		After("greet.go", diffAfter)
}

// golden compares got against testdata/<name>.golden, or rewrites it under
// -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if got != string(want) {
		t.Fatalf("output does not match %s:\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func TestRenderUnifiedGolden(t *testing.T) {
	for _, width := range []int{80, 120} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			golden(t, fmt.Sprintf("unified-%d", width), plainDiff().Render(width))
		})
	}
}

func TestRenderSplitGolden(t *testing.T) {
	for _, width := range []int{100, 140} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			golden(t, fmt.Sprintf("split-%d", width), plainDiff().RenderSplit(width))
		})
	}
}

// Below the floor two columns would be under 50 each, so RenderSplit must hand
// back the unified layout rather than a wall of truncations.
func TestRenderSplitFallsBackBelowMinWidth(t *testing.T) {
	width := SplitMinWidth - 1
	if got, want := plainDiff().RenderSplit(width), plainDiff().Render(width); got != want {
		t.Fatalf("RenderSplit(%d) did not fall back to unified:\n%s", width, got)
	}
}

func TestRenderCapsAtMaxLines(t *testing.T) {
	full := strings.Count(plainDiff().Render(80), "\n") + 1
	got := plainDiff().MaxLines(6).Render(80)
	lines := strings.Split(got, "\n")
	if len(lines) != 7 {
		t.Fatalf("MaxLines(6) rendered %d lines, want 7 (6 + tail):\n%s", len(lines), got)
	}
	if want := fmt.Sprintf("… +%d more lines", full-6); lines[6] != want {
		t.Fatalf("tail = %q, want %q", lines[6], want)
	}
	golden(t, "truncated", got)
}

// MaxLines above the output length must not add a tail claiming zero lines.
func TestMaxLinesAboveTheOutputLeavesItAlone(t *testing.T) {
	if got, want := plainDiff().MaxLines(500).Render(80), plainDiff().Render(80); got != want {
		t.Fatalf("MaxLines(500) changed the output:\n%s", got)
	}
}

// counting wraps a Highlighter to record how often it ran. The cache is
// otherwise invisible from outside: highlighting is the expensive per-line work
// a cache hit must skip entirely.
func countingHighlight(calls *int) Highlighter {
	return func(_, code string) string {
		*calls++
		return code
	}
}

func TestRenderReusesTheCacheAtTheSameWidth(t *testing.T) {
	calls := 0
	d := plainDiff().Highlighter(countingHighlight(&calls))
	first := d.Render(80)
	after := calls
	if after == 0 {
		t.Fatal("highlighter never ran on the first render")
	}
	for range 5 {
		if got := d.Render(80); got != first {
			t.Fatal("a cached render differs from the first one")
		}
	}
	if calls != after {
		t.Fatalf("highlighter ran %d more times on cached renders, want 0", calls-after)
	}
}

// Each layout and width is its own memo, and a builder mutation drops them all.
func TestCacheIsPerWidthAndLayoutAndDroppedOnMutation(t *testing.T) {
	calls := 0
	d := plainDiff().Highlighter(countingHighlight(&calls))
	d.Render(80)
	first := calls
	if d.Render(120); calls == first {
		t.Fatal("a new width served the cached render")
	}
	second := calls
	if d.RenderSplit(120); calls == second {
		t.Fatal("the split layout served the unified cache")
	}
	third := calls
	if d.ContextLines(1).Render(80); calls == third {
		t.Fatal("ContextLines did not invalidate the cache")
	}
}

// markStyle brackets whatever the emphasis style paints, so a test can see the
// changed spans without decoding escapes.
func markStyle(s lipgloss.Style, text string) string {
	if s.GetReverse() {
		return "«" + text + "»"
	}
	return text
}

// The point of the whole widget: on a modified pair, only the words that moved
// are emphasized.
func TestIntralineEmphasisMarksChangedWordsOnly(t *testing.T) {
	got := NewDiffView(DefaultTheme()).Painter(markStyle).Highlighter(nil).
		Before("a.go", "value := compute(alpha, beta)\n").
		After("a.go", "value := compute(gamma, beta)\n").
		Render(80)
	if !strings.Contains(got, "value := compute(«alpha», beta)") {
		t.Fatalf("removed line did not emphasize alpha alone:\n%s", got)
	}
	if !strings.Contains(got, "value := compute(«gamma», beta)") {
		t.Fatalf("added line did not emphasize gamma alone:\n%s", got)
	}
}

// A line with no counterpart has nothing to emphasize against, so it is
// highlighted instead and carries no emphasis at all.
func TestPureInsertionCarriesNoEmphasis(t *testing.T) {
	got := NewDiffView(DefaultTheme()).Painter(markStyle).Highlighter(nil).
		Before("a.go", "one\n").
		After("a.go", "one\ntwo\n").
		Render(80)
	if strings.Contains(got, "«") {
		t.Fatalf("an unpaired insertion was emphasized:\n%s", got)
	}
}

func TestIdenticalContentRendersNoChanges(t *testing.T) {
	d := NewDiffView(DefaultTheme()).Painter(Plain).Before("a.go", "x\n").After("a.go", "x\n")
	if got, want := d.Render(80), "no changes"; got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestRenderAtNonPositiveWidthIsEmpty(t *testing.T) {
	if got := plainDiff().Render(0); got != "" {
		t.Fatalf("Render(0) = %q, want empty", got)
	}
}

// A one-sided builder still has to label both headers, or a created file
// renders with a blank "---" line.
func TestHeadersFallBackToTheOtherSidesPath(t *testing.T) {
	got := NewDiffView(DefaultTheme()).Painter(Plain).Highlighter(nil).
		After("new.go", "package main\n").Render(80)
	if !strings.HasPrefix(got, "--- new.go\n+++ new.go\n") {
		t.Fatalf("headers = %q, want both sides labelled new.go", got)
	}
}

// Highlighting must color Go and must never fail: an extension chroma cannot
// place comes back as the text that went in.
func TestChromaHighlighterColorsGoAndDegradesQuietly(t *testing.T) {
	h := ChromaHighlighter(DefaultDiffSyntaxTheme)
	if got := h("main.go", "func main() {}"); !strings.Contains(got, "\x1b[") {
		t.Fatalf("Go was not highlighted: %q", got)
	}
	if got := h("notes.zzzz", "plain words"); !strings.Contains(got, "plain words") {
		t.Fatalf("unknown extension lost its text: %q", got)
	}
	if got := h("main.go", ""); got != "" {
		t.Fatalf("empty line = %q, want empty", got)
	}
}

// An unknown stylesheet must not leave chroma to resolve it to its own
// near-monochrome fallback.
func TestChromaHighlighterFallsBackOnAnUnknownStyle(t *testing.T) {
	if got := ChromaHighlighter("no-such-style")("main.go", "func main() {}"); !strings.Contains(got, "\x1b[") {
		t.Fatalf("unknown stylesheet produced unstyled output: %q", got)
	}
}

// Tabs cannot survive into the output: the split layout aligns by display
// width, which a tab does not have.
func TestTabsAreExpanded(t *testing.T) {
	got := NewDiffView(DefaultTheme()).Painter(Plain).Highlighter(nil).
		Before("a.go", "\tone\n").After("a.go", "\ttwo\n").Render(80)
	if strings.Contains(got, "\t") {
		t.Fatalf("output kept a tab: %q", got)
	}
}

// The left cell must be exactly the column width however long its line is, or
// the center gutter bends on the rows that hit the truncation cap.
func TestSplitGutterStaysAlignedOnLongLeftLines(t *testing.T) {
	long := "func aVeryLongFunctionNameThatKeepsGoingAndGoingAndGoingOnwards(x int) string {\n"
	got := NewDiffView(DefaultTheme()).Painter(Plain).Highlighter(nil).
		Before("a.go", long+"one\n").After("a.go", "func b() {}\none\n").
		RenderSplit(100)
	column := -1
	for _, line := range strings.Split(got, "\n") {
		i := strings.Index(line, splitGutter)
		if i < 0 {
			continue
		}
		width := lipgloss.Width(line[:i])
		if column == -1 {
			column = width
		}
		if width != column {
			t.Fatalf("gutter at column %d, want %d, in %q:\n%s", width, column, line, got)
		}
	}
	if column == -1 {
		t.Fatal("no split rows rendered")
	}
}

// An unknown stylesheet falls back to [DefaultDiffSyntaxTheme], not to chroma's
// own near-monochrome fallback — styles.Get resolves unknown names itself, so
// the nil check it never fails is not the guard.
func TestChromaHighlighterUnknownStyleMatchesTheDefault(t *testing.T) {
	code := "func main() {}"
	got := ChromaHighlighter("no-such-style")("main.go", code)
	want := ChromaHighlighter(DefaultDiffSyntaxTheme)("main.go", code)
	if got != want {
		t.Fatalf("unknown style = %q, want the default theme's %q", got, want)
	}
}

// Setters called with the value they already hold must keep the memo: a view
// function that threads MaxLines or ContextLines through every frame would
// otherwise re-diff and re-highlight on every repaint.
func TestUnchangedSettersKeepTheCache(t *testing.T) {
	calls := 0
	d := plainDiff().Highlighter(countingHighlight(&calls))
	d.MaxLines(6).ContextLines(3).Render(80)
	before := calls
	d.MaxLines(6).ContextLines(3).
		Before("greet.go", diffBefore).After("greet.go", diffAfter).
		Render(80)
	if calls != before {
		t.Fatalf("unchanged setters re-rendered: highlighter ran %d more times, want 0", calls-before)
	}
}

// Tab stops are display columns: a wide rune before a tab must not shift them.
func TestTabStopsCountDisplayColumns(t *testing.T) {
	if got, want := expandTabs("界\tx"), "界  x"; got != want {
		t.Fatalf("expandTabs = %q, want %q", got, want)
	}
	if got, want := expandTabs("é\tx"), "é   x"; got != want {
		t.Fatalf("expandTabs = %q, want %q", got, want)
	}
}

func TestMaxLinesTailIsSingularForOneHiddenLine(t *testing.T) {
	full := strings.Count(plainDiff().Render(80), "\n") + 1
	got := plainDiff().MaxLines(full - 1).Render(80)
	if !strings.HasSuffix(got, "… +1 more line") {
		t.Fatalf("tail = %q, want a singular line", got[strings.LastIndex(got, "\n")+1:])
	}
}

// Regression: wordsToRunes must hand back the tokens the closures collected.
// A go1.26 optimizer bug corrupted the named-return slice when both encode
// calls sat inside the return statement (see the comment in wordsToRunes):
// dependent modules saw segfaults and runaway allocations in markIntraline.
func TestWordsToRunesTokensStayIntact(t *testing.T) {
	a, b, tokens := wordsToRunes("old", "new")
	if len(a) != 1 || len(b) != 1 || len(tokens) != 2 {
		t.Fatalf("wordsToRunes = %v %v %v", a, b, tokens)
	}
	if tokens[a[0]-1] != "old" || tokens[b[0]-1] != "new" {
		t.Fatalf("token table corrupted: %q", tokens)
	}
}

// A one-line, one-word replacement is the smallest diff that walks the
// intraline pairing path end to end.
func TestSingleWordReplacementRenders(t *testing.T) {
	dv := NewDiffView(DefaultTheme()).Before("a.go", "old\n").After("a.go", "new\n")
	out := dv.Render(76)
	if !strings.Contains(out, "old") || !strings.Contains(out, "new") {
		t.Fatalf("render lost the changed words:\n%s", out)
	}
}
