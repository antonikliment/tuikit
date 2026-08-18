package main

import "testing"

// feed runs a sequence of gaps through the estimator, in ticks.
func feed(g *gapEstimator, gaps ...int) {
	for _, gap := range gaps {
		for range gap {
			g.Tick()
		}
		g.Settled()
	}
}

// A steady stream has no variation to cover, so the reserve settles on the gap
// itself rather than drifting up.
func TestReserveConvergesOnASteadyGap(t *testing.T) {
	var g gapEstimator
	for range 60 {
		feed(&g, 100)
	}
	if got := g.Reserve(); got < 90 || got > 130 {
		t.Fatalf("Reserve = %d after a steady 100-tick gap, want it close to 100", got)
	}
}

// The point of tracking deviation: a stream that alternates short and long gaps
// has to be covered for the long ones, not for the average.
func TestReserveCoversAVaryingGap(t *testing.T) {
	var steady, varying gapEstimator
	for range 60 {
		feed(&steady, 100)
		feed(&varying, 20, 180) // same mean, wildly different spread
	}
	if varying.Reserve() <= steady.Reserve() {
		t.Fatalf("Reserve = %d for a varying stream and %d for a steady one with the same mean, want more held for variation",
			varying.Reserve(), steady.Reserve())
	}
	if got := varying.Reserve(); got < 180 {
		t.Fatalf("Reserve = %d, want it to cover the long gap of 180", got)
	}
}

// A document that shortens has to stop holding a reserve it no longer needs, or
// the latency it bought never comes back.
func TestReserveFollowsTheStreamDownwards(t *testing.T) {
	var g gapEstimator
	for range 40 {
		feed(&g, 400)
	}
	high := g.Reserve()
	for range 40 {
		feed(&g, 20)
	}
	if g.Reserve() >= high {
		t.Fatalf("Reserve = %d after the stream sped up, want it below the earlier %d", g.Reserve(), high)
	}
}

// The startup case: nothing has settled yet, so there is no estimate. The
// drought so far stands in for one, which is what keeps the first block of an
// answer from stalling when it happens to be a fence.
func TestReserveUsesTheCurrentDroughtBeforeAnythingSettles(t *testing.T) {
	var g gapEstimator
	for range 250 {
		g.Tick()
	}
	if got := g.Reserve(); got != 250 {
		t.Fatalf("Reserve = %d before the first settlement, want the drought so far, 250", got)
	}
	g.Settled()
	if got := g.Waiting(); got != 0 {
		t.Fatalf("Waiting = %d just after a settlement, want 0", got)
	}
}

// Waiting is what separates a display that briefly caught up from one that has
// been frozen for seconds, so it has to count only the current drought.
func TestWaitingCountsOnlyTheCurrentDrought(t *testing.T) {
	var g gapEstimator
	feed(&g, 50, 50)
	for range 7 {
		g.Tick()
	}
	if got := g.Waiting(); got != 7 {
		t.Fatalf("Waiting = %d, want 7", got)
	}
}
