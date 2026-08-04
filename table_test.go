package tuikit_test

import (
	"testing"

	"github.com/antonikliment/tuikit"
)

func TestTableAlignsColumnsAndUppercasesHeader(t *testing.T) {
	got := tuikit.DefaultTheme().Table(tuikit.Plain,
		[]string{"name", "state"},
		[][]string{{"a", "running"}, {"longer", "idle"}},
		2)
	want := "NAME    STATE\na       running\nlonger  idle\n"
	if got != want {
		t.Errorf("Table = %q, want %q", got, want)
	}
}

func TestTableWidthIncludesHeader(t *testing.T) {
	got := tuikit.DefaultTheme().Table(tuikit.Plain,
		[]string{"profile", "n"}, [][]string{{"a", "1"}}, 1)
	want := "PROFILE N\na       1\n"
	if got != want {
		t.Errorf("Table = %q, want %q", got, want)
	}
}

func TestTableEmptyRows(t *testing.T) {
	if got := tuikit.DefaultTheme().Table(tuikit.Plain, []string{"a"}, nil, 2); got != "" {
		t.Errorf("Table with no rows = %q, want empty", got)
	}
}

func TestPairsPadsKeys(t *testing.T) {
	got := tuikit.DefaultTheme().Pairs(tuikit.Plain,
		[]string{"ngl", "context"}, []string{"auto", "4096"}, 2)
	want := "ngl      auto\ncontext  4096\n"
	if got != want {
		t.Errorf("Pairs = %q, want %q", got, want)
	}
}

func TestPairsMissingValue(t *testing.T) {
	got := tuikit.DefaultTheme().Pairs(tuikit.Plain, []string{"a"}, nil, 2)
	if want := "a\n"; got != want {
		t.Errorf("Pairs = %q, want %q", got, want)
	}
}
