package main

import (
	"slices"
	"testing"

	"github.com/antonikliment/tuikit/markdown"
)

// Every stylesheet this demo cycles has to be one the package accepts. An
// unknown name falls back to the default, so a rename in chroma would show up
// as a screen that does not change rather than as an error — which is the very
// failure the demo exists to make visible.
func TestCycledThemesAreAccepted(t *testing.T) {
	available := markdown.SyntaxThemes()
	for _, name := range themes {
		if !slices.Contains(available, name) {
			t.Errorf("unknown chroma stylesheet %q", name)
		}
	}
}

// Switching must actually repaint the code. The streamer caches on source and
// width, so reusing it across a switch would serve the previous palette back.
func TestSwitchingChangesTheRenderedCode(t *testing.T) {
	p := &syntaxPage{}
	seen := map[string]string{}
	for i := range themes {
		p.use(i)
		seen[themes[i]] = p.streamer.Render(sample, 60)
	}
	for i, a := range themes {
		for _, b := range themes[i+1:] {
			if seen[a] == seen[b] {
				t.Errorf("%q and %q rendered identically", a, b)
			}
		}
	}
}
