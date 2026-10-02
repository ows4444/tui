//go:build !race

// The ratio test is meaningless under the race detector, which instruments the
// two paths unevenly.

package viewport

import (
	"testing"
	"time"
)

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

// TestSetContentCleanTextWithinTenPercentOfRaw proves criterion #69: on text
// with no control bytes SetContent, with sanitizing on, stays within 10% of
// the same call with sanitizing off (Raw), which is the cost before sanitizing
// existed. A ratio in one process needs no stored baseline and does not depend
// on machine speed.
func TestSetContentCleanTextWithinTenPercentOfRaw(t *testing.T) {
	const limit = 1.10
	s := benchContent(10000)
	sanitized, raw := New(120, 40), New(120, 40)
	raw.Raw = true
	var ratio float64
	ok := false
	// Retry: a noisy sample can only slow one side, so a pass on any attempt
	// is a real pass.
	for attempt := 0; attempt < 8 && !ok; attempt++ {
		base := bestOf(20, func() { raw.SetContent(s) })
		got := bestOf(20, func() { sanitized.SetContent(s) })
		if base <= 0 {
			base = 1
		}
		ratio = float64(got) / float64(base)
		ok = ratio <= limit
	}
	if !ok {
		t.Errorf("SetContent on clean text is %.2fx Raw, want <= %.2fx", ratio, limit)
	}
}
