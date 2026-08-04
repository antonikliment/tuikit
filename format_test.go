package tuikit

import (
	"strings"
	"testing"
	"time"
)

func TestTransferredBytes(t *testing.T) {
	cases := []struct {
		name            string
		received, total int64
		want            string
	}{
		{"both known", 4 * 1024 * 1024 * 1024, 8 * 1024 * 1024 * 1024, "4.0 GiB / 8.0 GiB"},
		{"unknown total", 512, 0, "512 B"},
		{"negative total", 512, -1, "512 B"},
		{"nothing received yet", 0, 1024, "0 B / 1.0 KiB"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TransferredBytes(c.received, c.total); got != c.want {
				t.Fatalf("TransferredBytes(%d,%d) = %q, want %q", c.received, c.total, got, c.want)
			}
		})
	}
}

func TestCoarseDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want time.Duration
	}{
		{272914939 * time.Nanosecond, 272910 * time.Microsecond},
		{1234567 * time.Nanosecond, 1230 * time.Microsecond},
		{1234 * time.Nanosecond, 1230 * time.Nanosecond},
		{90 * time.Nanosecond, 90 * time.Nanosecond},
		{0, 0},
		{2500 * time.Millisecond, 2500 * time.Millisecond},
	}
	for _, c := range cases {
		if got := CoarseDuration(c.in); got != c.want {
			t.Fatalf("CoarseDuration(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

// Rounding by magnitude means a negative duration keeps its sign rather than
// being flipped or clamped.
func TestCoarseDurationNegative(t *testing.T) {
	got := CoarseDuration(-272914939 * time.Nanosecond)
	if got >= 0 {
		t.Fatalf("CoarseDuration(negative) = %v, want negative", got)
	}
	if want := -272910 * time.Microsecond; got != want {
		t.Fatalf("CoarseDuration = %v, want %v", got, want)
	}
}

// Coarsening must never change what a reader concludes: three significant
// figures is a display concern, not a measurement one.
func TestCoarseDurationStaysCloseToInput(t *testing.T) {
	for _, d := range []time.Duration{
		time.Nanosecond, 999 * time.Nanosecond, 5 * time.Microsecond,
		time.Millisecond, 1500 * time.Millisecond, 3 * time.Hour,
	} {
		got := CoarseDuration(d)
		if drift := (got - d).Abs(); drift*100 > d {
			t.Fatalf("CoarseDuration(%v) = %v drifts %v, more than 1%%", d, got, drift)
		}
	}
}

func TestAgeWithinWindow(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{"minutes", now.Add(-3 * time.Minute), "3m0s ago"},
		{"hours and minutes", now.Add(-(3*time.Hour + 23*time.Minute)), "3h23m0s ago"},
		{"just now", now, "0s ago"},
		{"sub-second rounds down", now.Add(-400 * time.Millisecond), "0s ago"},
		{"at the cutoff", now.Add(-AgeCutoff), "24h0m0s ago"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Age(c.at, now); got != c.want {
				t.Fatalf("Age = %q, want %q", got, c.want)
			}
		})
	}
}

// Outside the window the elapsed form stops being the useful one, so the
// absolute timestamp comes back instead.
func TestAgeOutsideWindowFallsBackToTimestamp(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name string
		at   time.Time
	}{
		{"older than the cutoff", now.Add(-AgeCutoff - time.Second)},
		{"in the future", now.Add(time.Hour)},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := Age(c.at, now)
			if strings.HasSuffix(got, "ago") {
				t.Fatalf("Age = %q, want an absolute timestamp", got)
			}
			parsed, err := time.Parse(time.RFC3339, got)
			if err != nil {
				t.Fatalf("Age = %q, not RFC 3339: %v", got, err)
			}
			if !parsed.Equal(c.at) {
				t.Fatalf("Age = %q, which is not the instant %v", got, c.at)
			}
		})
	}
}
