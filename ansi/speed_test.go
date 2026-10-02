//go:build !race

// The ratio test is meaningless under the race detector, which instruments
// the primitives far more heavily than the baseline byte loop.

package ansi

import (
	"strings"
	"testing"
	"time"
)

// speedSink keeps the compiler from discarding the measured work.
var speedSink int

// bestOf times fn over iters calls, seven times, and returns the fastest
// sample per call. The minimum discards scheduler and GC noise.
func bestOf(iters int, fn func()) time.Duration {
	best := time.Duration(1<<63 - 1)
	for s := 0; s < 7; s++ {
		start := time.Now()
		for i := 0; i < iters; i++ {
			fn()
		}
		if d := time.Since(start) / time.Duration(iters); d < best {
			best = d
		}
	}
	return best
}

// TestASCIISpeedRelativeToByteLoop guards the printable-ASCII fast path by
// ratio, so it needs no stored baseline and does not depend on machine speed:
// Width, Truncate and TrimLeftWidth over printable ASCII must stay within
// speedFactor of a plain byte loop over the same string, measured in the same
// process. Without the fast path Truncate is around 8x a byte loop.
// (internal/tools/benchgate gates allocations only; this test covers speed.)
func TestASCIISpeedRelativeToByteLoop(t *testing.T) {
	const speedFactor = 3.0
	s := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 12)
	w := len(s) - 1

	byteLoop := func() {
		n := 0
		for i := 0; i < len(s); i++ {
			if c := s[i]; c >= 0x20 && c < 0x7f {
				n++
			}
		}
		speedSink += n
	}
	cases := []struct {
		name string
		fn   func()
	}{
		{"Width", func() { speedSink += Width(s) }},
		{"Truncate", func() { speedSink += len(Truncate(s, w)) }},
		{"TrimLeftWidth", func() { speedSink += len(TrimLeftWidth(s, 1)) }},
	}
	const iters = 2000
	for _, c := range cases {
		var ratio float64
		ok := false
		// Retry: a noisy sample can only slow the primitive or speed the
		// baseline, so a pass on any attempt is a real pass.
		for attempt := 0; attempt < 5 && !ok; attempt++ {
			base := bestOf(iters, byteLoop)
			got := bestOf(iters, c.fn)
			if base <= 0 {
				base = 1
			}
			ratio = float64(got) / float64(base)
			ok = ratio <= speedFactor
		}
		if !ok {
			t.Errorf("%s on printable ASCII is %.1fx a byte loop, want <= %.1fx", c.name, ratio, speedFactor)
		}
	}
}
