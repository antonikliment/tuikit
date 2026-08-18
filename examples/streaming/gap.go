package main

// How long the display waits between settlements, and how much that varies.
//
// The reveal is a playout buffer: a bursty producer (blocks settling) feeding a
// constant-rate consumer (the display clock). Sizing the buffer is the same
// problem a VoIP jitter buffer solves, and the answer there is not to measure
// the current gap but to track a running estimate of the gap and its variation,
// then hold enough to cover a high percentile of it. RFC 3550 estimates arrival
// jitter this way; this is the same exponentially weighted pair, with the
// samples being settle gaps rather than packet transit times.
//
// It is deliberately not a predictor. It does not know a fence is coming, only
// how far apart settlements have been running — which is enough, because the
// buffer is held all the time and only spent when a gap actually opens.

// gain is how fast the estimate follows the stream. RFC 3550 uses 1/16 for
// jitter over hundreds of packets a second; a single answer settles a few dozen
// blocks in total, so this has to converge in far fewer samples.
const gain = 1.0 / 8.0

// deviations is how many deviations above the mean to hold. Four covers the
// tail of a skewed distribution — and settle gaps are badly skewed, since a
// document is mostly short paragraphs with the occasional long fence.
const deviations = 4

type gapEstimator struct {
	mean, deviation float64
	since           int // ticks since the last settlement
	seen            int
}

// Tick records one display tick with nothing new settled.
func (g *gapEstimator) Tick() { g.since++ }

// Settled records that a block settled, folding the gap that just ended into
// the estimate. Gaps are measured between settlements rather than sampled on a
// clock, so a long drought contributes one large sample instead of many small
// ones — the same reason jitter buffers adapt at talkspurt boundaries rather
// than continuously.
func (g *gapEstimator) Settled() {
	gap := float64(g.since)
	g.since, g.seen = 0, g.seen+1
	if g.seen == 1 {
		g.mean = gap
		return
	}
	g.deviation += gain * (abs(gap-g.mean) - g.deviation)
	g.mean += gain * (gap - g.mean)
}

// Reserve is how many ticks of drought to be ready for: the mean gap plus four
// deviations of it.
//
// Before anything has settled there is no estimate to work from, so the current
// drought stands in for one. That matters at the top of an answer, where the
// first block can be a fence and getting it wrong means stalling on the very
// first thing the reader sees — the startup case that buffer-based rate
// adaptation also handles separately from steady state.
func (g *gapEstimator) Reserve() int {
	if g.seen == 0 {
		return g.since
	}
	return int(g.mean + deviations*g.deviation)
}

// Waiting is how long the current drought has been running. The policy reads it
// to tell a display that has briefly caught up from one that has been frozen
// for seconds.
func (g *gapEstimator) Waiting() int { return g.since }

func (g *gapEstimator) reset() { *g = gapEstimator{} }

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
