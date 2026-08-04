package tuikit

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestColumnsMeasuresWidestCell(t *testing.T) {
	widths := Columns([][]string{
		{"NAME", "STATE"},
		{"a-long-profile", "ok"},
		{"short", "running"},
	})
	if want := []int{14, 7}; len(widths) != 2 || widths[0] != want[0] || widths[1] != want[1] {
		t.Fatalf("Columns = %v, want %v", widths, want)
	}
}

// A ragged table must not panic and must size itself to the longest row.
func TestColumnsRaggedRows(t *testing.T) {
	widths := Columns([][]string{{"a"}, {"bb", "ccc"}, nil})
	if want := []int{2, 3}; len(widths) != 2 || widths[0] != want[0] || widths[1] != want[1] {
		t.Fatalf("Columns = %v, want %v", widths, want)
	}
	if got := Columns(nil); len(got) != 0 {
		t.Fatalf("Columns(nil) = %v, want empty", got)
	}
}

// The reason this package owns the helper at all: a styled cell carries escape
// sequences that occupy no screen columns, and measuring with len skews the
// whole table right.
func TestColumnsIgnoresANSI(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("ok")
	if len(styled) <= len("ok") {
		t.Skip("lipgloss emitted no escapes; nothing to measure around")
	}
	widths := Columns([][]string{{styled}, {"okay"}})
	if widths[0] != 4 {
		t.Fatalf("width = %d, want 4 (styled %q must measure as 2)", widths[0], styled)
	}
}

func TestColumnsCountsWideRunesAsTwoCells(t *testing.T) {
	if got := Columns([][]string{{"世界"}})[0]; got != 4 {
		t.Fatalf("width of 世界 = %d, want 4", got)
	}
}

func TestJoinCellsAligns(t *testing.T) {
	widths := []int{5, 3}
	got := JoinCells([]string{"ab", "cd", "e"}, widths, 2)
	if want := "ab     cd   e"; got != want {
		t.Fatalf("JoinCells = %q, want %q", got, want)
	}
	// Every row lines the second column up at the same offset.
	first := JoinCells([]string{"ab", "x"}, widths, 2)
	second := JoinCells([]string{"abcde", "y"}, widths, 2)
	if strings.Index(first, "x") != strings.Index(second, "y") {
		t.Fatalf("columns misaligned: %q vs %q", first, second)
	}
}

func TestJoinCellsNoTrailingWhitespace(t *testing.T) {
	got := JoinCells([]string{"ab", ""}, []int{5, 9}, 2)
	if got != strings.TrimRight(got, " ") {
		t.Fatalf("JoinCells left trailing whitespace: %q", got)
	}
}

// A row longer than the measured widths must still render every cell.
func TestJoinCellsRowLongerThanWidths(t *testing.T) {
	got := JoinCells([]string{"a", "b", "c"}, []int{1}, 1)
	if !strings.Contains(got, "b") || !strings.Contains(got, "c") {
		t.Fatalf("JoinCells dropped cells: %q", got)
	}
}

func TestJoinCellsEdgeCases(t *testing.T) {
	if got := JoinCells(nil, []int{3}, 2); got != "" {
		t.Fatalf("JoinCells(nil) = %q, want empty", got)
	}
	if got := JoinCells([]string{"a", "b"}, []int{1, 1}, -5); got != "ab" {
		t.Fatalf("negative gap = %q, want %q", got, "ab")
	}
}

// Aligning by display width is the whole point: a styled cell must occupy the
// same screen columns as its plain equivalent.
func TestJoinCellsAlignsStyledCells(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("ok")
	widths := Columns([][]string{{styled, "x"}, {"plain", "y"}})
	row := ansi.Strip(JoinCells([]string{styled, "x"}, widths, 2))
	plain := JoinCells([]string{"ok", "x"}, widths, 2)
	if row != plain {
		t.Fatalf("styled row %q does not lay out like plain %q", row, plain)
	}
}

func TestPad(t *testing.T) {
	if got := Pad("ab", 5); got != "ab   " {
		t.Fatalf("Pad = %q, want %q", got, "ab   ")
	}
	// Pad never truncates.
	if got := Pad("abcdef", 3); got != "abcdef" {
		t.Fatalf("Pad truncated: %q", got)
	}
	if got := ansi.StringWidth(Pad(lipgloss.NewStyle().Bold(true).Render("ab"), 5)); got != 5 {
		t.Fatalf("padded display width = %d, want 5", got)
	}
}

func TestWidest(t *testing.T) {
	if got := Widest([]string{"a", "abc", "ab"}); got != 3 {
		t.Fatalf("Widest = %d, want 3", got)
	}
	if got := Widest(nil); got != 0 {
		t.Fatalf("Widest(nil) = %d, want 0", got)
	}
}

func TestIndent(t *testing.T) {
	if got := Indent("x", 2); got != "    x" {
		t.Fatalf("Indent = %q, want %q", got, "    x")
	}
	for _, depth := range []int{0, -1} {
		if got := Indent("x", depth); got != "x" {
			t.Fatalf("Indent(x,%d) = %q, want %q", depth, got, "x")
		}
	}
}

func TestIndentLines(t *testing.T) {
	if got := IndentLines("a\nb", 1); got != "  a\n  b\n" {
		t.Fatalf("IndentLines = %q", got)
	}
	// A newline-terminated block does not accumulate blank lines when nested.
	if got := IndentLines("a\n", 1); got != "  a\n" {
		t.Fatalf("IndentLines kept a trailing blank line: %q", got)
	}
	for _, in := range []string{"", "\n\n"} {
		if got := IndentLines(in, 1); got != "" {
			t.Fatalf("IndentLines(%q) = %q, want empty", in, got)
		}
	}
	// Interior blank lines are preserved, and carry no trailing indent.
	if got := IndentLines("a\n\nb", 1); got != "  a\n  \n  b\n" {
		t.Fatalf("IndentLines dropped an interior blank line: %q", got)
	}
}
