package tuikit

import "testing"

func TestPageDigit(t *testing.T) {
	cases := []struct {
		in  string
		idx int
		ok  bool
	}{
		{"1", 0, true},
		{"2", 1, true},
		{"9", 8, true},
		{"0", 0, false},
		{"a", 0, false},
		{"12", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		idx, ok := pageDigit(c.in)
		if ok != c.ok || (ok && idx != c.idx) {
			t.Fatalf("pageDigit(%q) = (%d, %v), want (%d, %v)", c.in, idx, ok, c.idx, c.ok)
		}
	}
}

func TestDefaultKeyMapQuitIsCtrlCOnly(t *testing.T) {
	keys := DefaultKeyMap()
	if !keys.Quit.Enabled() {
		t.Fatal("quit binding should be enabled")
	}
	if got := keys.Quit.Keys(); len(got) != 1 || got[0] != "ctrl+c" {
		t.Fatalf("quit keys = %v, want [ctrl+c]", got)
	}
}
