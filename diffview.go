package tuikit

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffView renders a diff between two versions of a file, unified or
// side-by-side, syntax-highlighted and with the changed spans inside a modified
// line pair picked out.
//
// A raw +/- diff is the hardest artifact a TUI produces to read: no color, no
// intraline emphasis, and a renamed identifier hides in a wall of otherwise
// identical text. DiffView is the readable form of the same data, driven by a
// [Theme] like the rest of the kit.
//
// It renders a string and holds no scroll state — the caller owns the viewport
// — and it does no file I/O; content is passed in.
//
//	out := NewDiffView(theme).
//		Before("main.go", old).
//		After("main.go", new).
//		ContextLines(3).
//		Render(width)
//
// Diffing and highlighting are O(file) and a TUI repaints every frame, so
// results are memoized per (width, layout) and the memo is dropped on any
// builder mutation. A DiffView is therefore not safe for concurrent use.
type DiffView struct {
	theme       Theme
	paint       Painter
	highlight   Highlighter
	beforePath  string
	before      string
	afterPath   string
	after       string
	contextLine int
	maxLines    int

	// lines is the diff itself: computed once per content change and shared by
	// both layouts, since neither changes what the diff is.
	lines []diffLine
	// cache holds rendered output per layout and width. Both it and lines are
	// dropped by every builder setter, so a mutated builder cannot serve stale
	// output.
	cache map[diffKey]string
}

// diffKey identifies one rendered result: the layout and the width it was laid
// out at. Content is not part of the key because changing it clears the cache.
type diffKey struct {
	width int
	split bool
}

// Highlighter colors one line of code for display, keyed by the filename it
// came from. It must never fail: a line it cannot lex is returned as it came
// in, because an unhighlightable file must still be readable.
//
// Its result carries the highlighter's own SGR sequences — see [RenderFunc] for
// why that means not wrapping it in a foreground style.
type Highlighter func(filename, code string) string

// DefaultDiffSyntaxTheme is the chroma stylesheet [ChromaHighlighter] uses when
// given an empty name, matching the markdown package's default so a program
// using both does not show two palettes.
const DefaultDiffSyntaxTheme = "tokyonight-night"

// SplitMinWidth is the narrowest total width [DiffView.RenderSplit] will lay
// two columns out in. Below it each half is under 50 columns, which turns every
// line of real code into a truncation, so RenderSplit falls back to the unified
// layout instead of producing something unreadable.
const SplitMinWidth = 100

// splitGutter separates the two columns of the side-by-side layout.
const splitGutter = " │ "

// NewDiffView returns a DiffView drawing from theme, painting unconditionally
// ([Paint]) and highlighting with [ChromaHighlighter], with three lines of
// context and no line cap.
func NewDiffView(theme Theme) *DiffView {
	return &DiffView{
		theme:       theme,
		paint:       Paint,
		highlight:   ChromaHighlighter(DefaultDiffSyntaxTheme),
		contextLine: 3,
		cache:       map[diffKey]string{},
	}
}

// Before sets the original path and content.
func (d *DiffView) Before(path, content string) *DiffView {
	d.beforePath, d.before = path, content
	return d.invalidate()
}

// After sets the new path and content.
func (d *DiffView) After(path, content string) *DiffView {
	d.afterPath, d.after = path, content
	return d.invalidate()
}

// ContextLines sets how many unchanged lines are kept either side of a change.
// A negative count means zero; a count at or above the file length shows the
// whole file as one hunk.
func (d *DiffView) ContextLines(n int) *DiffView {
	d.contextLine = max(0, n)
	return d.invalidate()
}

// MaxLines caps the output at n rendered lines, replacing the remainder with a
// "… +N more lines" tail so a caller can show a preview and expand later. Zero
// or less means no cap.
func (d *DiffView) MaxLines(n int) *DiffView {
	d.maxLines = n
	return d.invalidate()
}

// Painter sets how styles are applied — pass [Plain] to render without escapes.
func (d *DiffView) Painter(p Painter) *DiffView {
	if p == nil {
		p = Plain
	}
	d.paint = p
	return d.invalidate()
}

// Highlighter sets the syntax highlighter. A nil highlighter disables
// highlighting, which is what tests and plain-text destinations want.
func (d *DiffView) Highlighter(h Highlighter) *DiffView {
	d.highlight = h
	return d.invalidate()
}

// invalidate drops everything derived from the builder's inputs.
func (d *DiffView) invalidate() *DiffView {
	d.lines = nil
	clear(d.cache)
	return d
}

// Render lays the diff out as a unified diff at width.
func (d *DiffView) Render(width int) string { return d.render(width, false) }

// RenderSplit lays the diff out side by side: old on the left, new on the
// right, aligned row by row. Below [SplitMinWidth] it renders unified instead.
func (d *DiffView) RenderSplit(width int) string { return d.render(width, true) }

// render serves the memo for (width, layout), computing it on a miss. A
// side-by-side request narrower than [SplitMinWidth] is served the unified
// layout, and shares its memo entry.
func (d *DiffView) render(width int, wantSplit bool) string {
	if width <= 0 {
		return ""
	}
	split := wantSplit && width >= SplitMinWidth
	if d.cache == nil {
		d.cache = map[diffKey]string{}
	}
	key := diffKey{width: width, split: split}
	if out, ok := d.cache[key]; ok {
		return out
	}
	rows := d.rows(width, split)
	if d.maxLines > 0 && len(rows) > d.maxLines {
		hidden := len(rows) - d.maxLines
		rows = append(rows[:d.maxLines:d.maxLines],
			d.paint(d.theme.MutedStyle(), fmt.Sprintf("… +%d more lines", hidden)))
	}
	out := strings.Join(rows, "\n")
	d.cache[key] = out
	return out
}

// rows renders the header plus every hunk in the requested layout.
func (d *DiffView) rows(width int, split bool) []string {
	lines := d.diff()
	if len(lines) == 0 {
		return []string{d.paint(d.theme.MutedStyle(), "no changes")}
	}
	rows := []string{
		d.paint(d.theme.Accent(d.theme.Red), "--- "+d.pathOr(d.beforePath, d.afterPath)),
		d.paint(d.theme.Accent(d.theme.Green), "+++ "+d.pathOr(d.afterPath, d.beforePath)),
	}
	numWidth := digits(len(lines))
	for _, h := range hunks(lines, d.contextLine) {
		rows = append(rows, d.paint(d.theme.Accent(d.theme.Cyan), h.header()))
		if split {
			rows = append(rows, d.splitRows(h.lines, numWidth, width)...)
			continue
		}
		for _, line := range h.lines {
			rows = append(rows, d.unifiedRow(line, numWidth))
		}
	}
	return rows
}

// pathOr falls back to the other side's path, so a one-sided builder (a created
// or deleted file) still labels both headers.
func (d *DiffView) pathOr(path, fallback string) string {
	if path != "" {
		return path
	}
	if fallback != "" {
		return fallback
	}
	return "(unnamed)"
}

// diff computes the line diff once per content change.
func (d *DiffView) diff() []diffLine {
	if d.lines == nil {
		d.lines = diffLines(d.before, d.after)
	}
	return d.lines
}

// unifiedRow renders one line as "old new ±content".
func (d *DiffView) unifiedRow(line diffLine, numWidth int) string {
	style := d.style(line.kind)
	gutter := fmt.Sprintf("%s %s %s", num(line.oldNum, numWidth), num(line.newNum, numWidth), line.sign())
	return strings.TrimRight(d.paint(style, gutter)+" "+d.content(line, style), " ")
}

// splitRows lays a hunk out as two aligned columns. Deleted and inserted runs
// are zipped so a modified pair shares a row, which is what makes the intraline
// emphasis on either side line up.
func (d *DiffView) splitRows(lines []diffLine, numWidth, width int) []string {
	column := (width - ansi.StringWidth(splitGutter)) / 2
	var rows []string
	for i := 0; i < len(lines); {
		if lines[i].kind == kindEqual {
			rows = append(rows, d.splitRow(&lines[i], &lines[i], numWidth, column))
			i++
			continue
		}
		dels := run(lines, i, kindDelete)
		ins := run(lines, i+dels, kindInsert)
		left, right := lines[i:i+dels], lines[i+dels:i+dels+ins]
		for row := range max(len(left), len(right)) {
			rows = append(rows, d.splitRow(at(left, row), at(right, row), numWidth, column))
		}
		i += dels + ins
	}
	return rows
}

// splitRow renders one row of the side-by-side layout; either side may be nil,
// which renders as blank padding so the columns stay aligned.
func (d *DiffView) splitRow(left, right *diffLine, numWidth, column int) string {
	row := d.cell(left, numWidth, column, true) + splitGutter + d.cell(right, numWidth, column, false)
	return strings.TrimRight(row, " ")
}

// cell renders one side of a side-by-side row, padded and hard-truncated to the
// column width. old picks which line number the cell carries.
func (d *DiffView) cell(line *diffLine, numWidth, column int, old bool) string {
	if line == nil {
		return strings.Repeat(" ", max(0, column))
	}
	style := d.style(line.kind)
	number := line.newNum
	if old {
		number = line.oldNum
	}
	gutter := d.paint(style, num(number, numWidth)+" "+line.sign())
	body := ansi.Truncate(d.content(*line, style), max(0, column-numWidth-2), "…")
	return Pad(gutter+" "+body, column)
}

// content renders a line's text: its changed spans emphasized when it is half
// of a modified pair, otherwise syntax-highlighted.
//
// The two are exclusive on purpose. Emphasis has to style substrings, and
// splicing a style into a chroma-colored string means editing SGR state mid-run
// — so on the lines that have a counterpart, emphasis wins, because picking the
// renamed identifier out of an otherwise identical line is the whole point.
// Every other line, which in a normal hunk is most of them, is highlighted.
func (d *DiffView) content(line diffLine, style lipgloss.Style) string {
	if len(line.spans) > 0 {
		var b strings.Builder
		for _, s := range line.spans {
			if s.changed {
				b.WriteString(d.paint(style.Bold(true).Reverse(true), s.text))
				continue
			}
			b.WriteString(d.paint(style, s.text))
		}
		return b.String()
	}
	if d.highlight != nil {
		return d.highlight(d.pathOr(d.afterPath, d.beforePath), line.text)
	}
	return d.paint(style, line.text)
}

// style is the palette role a line kind is drawn in.
func (d *DiffView) style(kind diffKind) lipgloss.Style {
	switch kind {
	case kindInsert:
		return d.theme.Accent(d.theme.Green)
	case kindDelete:
		return d.theme.Accent(d.theme.Red)
	default:
		return d.theme.MutedStyle()
	}
}

// diffKind is what happened to a line.
type diffKind int

const (
	kindEqual diffKind = iota
	kindDelete
	kindInsert
)

// diffLine is one line of the diff, numbered on the sides it exists on (zero
// where it does not) and optionally carrying intraline spans.
type diffLine struct {
	kind           diffKind
	text           string
	oldNum, newNum int
	spans          []diffSpan
}

// sign is the unified-diff marker for the line.
func (l diffLine) sign() string {
	switch l.kind {
	case kindInsert:
		return "+"
	case kindDelete:
		return "-"
	default:
		return " "
	}
}

// diffSpan is a run of a line marked as changed or not by the word diff.
type diffSpan struct {
	text    string
	changed bool
}

// diffLines diffs before against after line by line, numbers the result, and
// marks the changed words inside modified line pairs.
func diffLines(before, after string) []diffLine {
	if before == after {
		return nil
	}
	dmp := diffmatchpatch.New()
	a, b, index := dmp.DiffLinesToRunes(before, after)
	diffs := dmp.DiffCharsToLines(dmp.DiffMainRunes(a, b, false), index)

	var lines []diffLine
	oldNum, newNum := 0, 0
	for _, d := range diffs {
		for _, text := range splitLines(d.Text) {
			line := diffLine{text: text}
			switch d.Type {
			case diffmatchpatch.DiffInsert:
				newNum++
				line.kind, line.newNum = kindInsert, newNum
			case diffmatchpatch.DiffDelete:
				oldNum++
				line.kind, line.oldNum = kindDelete, oldNum
			default:
				oldNum, newNum = oldNum+1, newNum+1
				line.oldNum, line.newNum = oldNum, newNum
			}
			lines = append(lines, line)
		}
	}
	markIntraline(lines)
	return lines
}

// splitLines splits a chunk into lines without inventing a trailing empty one,
// expanding tabs on the way. Tabs have to go: a diff aligns columns by display
// width, and a tab's width depends on where the terminal thinks the tab stops
// are — which, once a line number gutter and a split column sit to its left, is
// nowhere the renderer can predict.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, line := range lines {
		lines[i] = expandTabs(line)
	}
	return lines
}

// TabWidth is how many spaces [DiffView] expands a leading or embedded tab to.
const TabWidth = 4

// expandTabs replaces tabs with spaces up to the next multiple of [TabWidth].
func expandTabs(line string) string {
	if !strings.Contains(line, "\t") {
		return line
	}
	var b strings.Builder
	for _, r := range line {
		if r != '\t' {
			b.WriteRune(r)
			continue
		}
		b.WriteString(strings.Repeat(" ", TabWidth-b.Len()%TabWidth))
	}
	return b.String()
}

// markIntraline pairs each deleted line with the inserted line at the same
// offset in the following run and marks the words that differ between them.
// Unpaired lines — a pure insertion, a deletion with no replacement — are left
// alone: there is no counterpart for a word to differ from.
func markIntraline(lines []diffLine) {
	for i := 0; i < len(lines); {
		dels := run(lines, i, kindDelete)
		if dels == 0 {
			i++
			continue
		}
		ins := run(lines, i+dels, kindInsert)
		for pair := range min(dels, ins) {
			left, right := &lines[i+pair], &lines[i+dels+pair]
			left.spans, right.spans = wordSpans(left.text, right.text)
		}
		i += dels + ins
	}
}

// run counts how many consecutive lines from i have the given kind.
func run(lines []diffLine, i int, kind diffKind) int {
	n := 0
	for i+n < len(lines) && lines[i+n].kind == kind {
		n++
	}
	return n
}

// at returns the line at index, or nil past the end — the blank half of an
// unbalanced side-by-side row.
func at(lines []diffLine, index int) *diffLine {
	if index >= len(lines) {
		return nil
	}
	return &lines[index]
}

// wordSpans diffs two lines at word granularity and returns the spans of each,
// changed words flagged. Diffing words rather than characters is what keeps the
// emphasis on the identifier that was renamed instead of on the three letters
// it happens to share with the old one.
func wordSpans(before, after string) (left, right []diffSpan) {
	a, b, tokens := wordsToRunes(before, after)
	dmp := diffmatchpatch.New()
	for _, d := range dmp.DiffMainRunes(a, b, false) {
		var text strings.Builder
		for _, r := range d.Text {
			text.WriteString(tokens[int(r)-1])
		}
		span := diffSpan{text: text.String(), changed: d.Type != diffmatchpatch.DiffEqual}
		if d.Type != diffmatchpatch.DiffInsert {
			left = append(left, span)
		}
		if d.Type != diffmatchpatch.DiffDelete {
			right = append(right, span)
		}
	}
	return left, right
}

// wordsToRunes tokenizes both lines and encodes each distinct token as a rune,
// so the diff runs over words at the cost of a character diff.
func wordsToRunes(before, after string) (a, b []rune, tokens []string) {
	seen := map[string]rune{}
	encode := func(line string) []rune {
		var out []rune
		for _, token := range words(line) {
			r, ok := seen[token]
			if !ok {
				tokens = append(tokens, token)
				r = rune(len(tokens))
				seen[token] = r
			}
			out = append(out, r)
		}
		return out
	}
	return encode(before), encode(after), tokens
}

// words splits a line into identifier-ish runs and single other runes, so
// punctuation forms its own boundaries.
func words(line string) []string {
	var out []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			out = append(out, word.String())
			word.Reset()
		}
	}
	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			word.WriteRune(r)
			continue
		}
		flush()
		out = append(out, string(r))
	}
	flush()
	return out
}

// diffHunk is a contiguous stretch of the diff worth showing: the changes plus
// their context.
type diffHunk struct{ lines []diffLine }

// header is the hunk's "@@ -a,b +c,d @@" line.
func (h diffHunk) header() string {
	oldStart, oldCount, newStart, newCount := 0, 0, 0, 0
	for _, line := range h.lines {
		if line.oldNum > 0 {
			oldCount++
			if oldStart == 0 {
				oldStart = line.oldNum
			}
		}
		if line.newNum > 0 {
			newCount++
			if newStart == 0 {
				newStart = line.newNum
			}
		}
	}
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
}

// hunks keeps every changed line plus context lines either side of it, and
// groups the survivors into contiguous stretches.
func hunks(lines []diffLine, context int) []diffHunk {
	keep := make([]bool, len(lines))
	for i, line := range lines {
		if line.kind == kindEqual {
			continue
		}
		for j := max(0, i-context); j <= min(len(lines)-1, i+context); j++ {
			keep[j] = true
		}
	}
	var out []diffHunk
	for i := 0; i < len(lines); i++ {
		if !keep[i] {
			continue
		}
		start := i
		for i < len(lines) && keep[i] {
			i++
		}
		out = append(out, diffHunk{lines: lines[start:i]})
	}
	return out
}

// num right-aligns a line number in width columns, blank for the side a line
// does not exist on.
func num(n, width int) string {
	if n == 0 {
		return strings.Repeat(" ", width)
	}
	return fmt.Sprintf("%*d", width, n)
}

// digits is how many columns a line number up to n needs.
func digits(n int) int {
	return len(fmt.Sprint(max(1, n)))
}

// ChromaHighlighter returns a [Highlighter] that colors a line with chroma,
// picking the lexer from the filename. An unknown stylesheet or an unlexable
// file degrades to plain text rather than to an error.
//
// Lines are lexed one at a time, since a diff shows lines out of order and a
// hunk has no whole-file context to lex against. The cost is that a line inside
// a multi-line string or comment is colored as if it were code — visible, but
// far cheaper than re-lexing both files for every frame.
//
// The returned function caches its lexer per filename and is safe for
// concurrent use.
func ChromaHighlighter(styleName string) Highlighter {
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Get(DefaultDiffSyntaxTheme)
	}
	formatter := formatters.Get("terminal256")
	var mu sync.Mutex
	cache := map[string]chroma.Lexer{}

	return func(filename, code string) string {
		if code == "" {
			return code
		}
		mu.Lock()
		lexer, ok := cache[filename]
		if !ok {
			lexer = lexers.Match(filename)
			if lexer == nil {
				lexer = lexers.Fallback
			}
			lexer = chroma.Coalesce(lexer)
			cache[filename] = lexer
		}
		mu.Unlock()

		iterator, err := lexer.Tokenise(nil, code)
		if err != nil {
			return code
		}
		var b strings.Builder
		if err := formatter.Format(&b, style, iterator); err != nil {
			return code
		}
		return strings.TrimRight(b.String(), "\n")
	}
}
