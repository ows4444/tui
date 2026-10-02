package motion

import (
	"math"
	"testing"
	"time"
)

var eases = map[string]struct {
	fn        Ease
	monotonic bool
}{
	"Linear": {Linear, true}, "InQuad": {InQuad, true}, "OutQuad": {OutQuad, true},
	"InOutQuad": {InOutQuad, true}, "InCubic": {InCubic, true}, "OutCubic": {OutCubic, true},
	"InOutCubic": {InOutCubic, true}, "OutBack": {OutBack, false},
}

func TestEaseEndpoints(t *testing.T) {
	for name, e := range eases {
		if got := e.fn(0); math.Abs(got) > 1e-12 {
			t.Errorf("%s(0) = %v, want 0", name, got)
		}
		if got := e.fn(1); math.Abs(got-1) > 1e-12 {
			t.Errorf("%s(1) = %v, want 1", name, got)
		}
	}
}

func TestEaseClampsOutOfRangeInput(t *testing.T) {
	for name, e := range eases {
		for _, in := range []float64{-5, -0.001, 1.001, 99, math.Inf(1), math.Inf(-1), math.NaN()} {
			got := e.fn(in)
			want := 0.0
			if in > 1 {
				want = 1
			}
			if math.Abs(got-want) > 1e-12 {
				t.Errorf("%s(%v) = %v, want %v", name, in, got, want)
			}
		}
	}
}

func TestEaseMonotonicWhereDocumented(t *testing.T) {
	for name, e := range eases {
		prev := e.fn(0)
		rose, fell := false, false
		for i := 1; i <= 1000; i++ {
			v := e.fn(float64(i) / 1000)
			if v < prev-1e-12 {
				fell = true
			}
			if v > 1+1e-12 {
				rose = true
			}
			prev = v
		}
		if e.monotonic && (fell || rose) {
			t.Errorf("%s is documented monotonic but fell=%v overshot=%v", name, fell, rose)
		}
		if !e.monotonic && !rose {
			t.Errorf("%s is documented as overshooting but never exceeded 1", name)
		}
	}
}

func TestEaseShapes(t *testing.T) {
	// In-eases start slower than linear, out-eases faster, at the midpoint.
	if !(InQuad(0.5) < 0.5 && InCubic(0.5) < InQuad(0.5)) {
		t.Error("InQuad/InCubic should lag Linear at 0.5")
	}
	if !(OutQuad(0.5) > 0.5 && OutCubic(0.5) > OutQuad(0.5)) {
		t.Error("OutQuad/OutCubic should lead Linear at 0.5")
	}
	if math.Abs(InOutQuad(0.5)-0.5) > 1e-12 || math.Abs(InOutCubic(0.5)-0.5) > 1e-12 {
		t.Error("InOut eases should pass through the midpoint")
	}
}

func TestTweenInterpolatesAndClamps(t *testing.T) {
	tw := Tween{From: 10, To: 30, Duration: time.Second}
	for _, c := range []struct {
		at   time.Duration
		want float64
	}{
		{-time.Second, 10}, {0, 10}, {500 * time.Millisecond, 20}, {time.Second, 30}, {5 * time.Second, 30},
	} {
		if got := tw.At(c.at); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("At(%v) = %v, want %v", c.at, got, c.want)
		}
	}
	// The Ease shapes the middle but not the ends.
	q := Tween{From: 0, To: 100, Duration: time.Second, Ease: InQuad}
	if got := q.At(500 * time.Millisecond); math.Abs(got-25) > 1e-9 {
		t.Errorf("InQuad midpoint = %v, want 25", got)
	}
	// A backwards tween works.
	if got := (Tween{From: 5, To: 1, Duration: time.Second}).At(500 * time.Millisecond); math.Abs(got-3) > 1e-9 {
		t.Errorf("reverse midpoint = %v, want 3", got)
	}
}

func TestTweenAtOrPastDurationIsExactlyTo(t *testing.T) {
	// OutBack would otherwise leave float error or an overshoot at the end.
	tw := Tween{From: 0, To: 7.5, Duration: 300 * time.Millisecond, Ease: OutBack}
	if got := tw.At(300 * time.Millisecond); got != 7.5 {
		t.Errorf("At(Duration) = %v, want exactly 7.5", got)
	}
	if got := tw.At(time.Hour); got != 7.5 {
		t.Errorf("At(1h) = %v, want exactly 7.5", got)
	}
}

func TestTweenReducedMotionJumpsToTheEnd(t *testing.T) {
	tw := Tween{From: 0, To: 42, Duration: time.Second, Ease: OutQuad, Motion: Reduced}
	for _, at := range []time.Duration{-time.Second, 0, 100 * time.Millisecond, 999 * time.Millisecond} {
		if got := tw.At(at); got != 42 {
			t.Errorf("reduced At(%v) = %v, want 42", at, got)
		}
		if !tw.Done(at) {
			t.Errorf("reduced Done(%v) = false, want true", at)
		}
	}
}

func TestTweenWithoutADurationIsInstant(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		tw := Tween{From: 1, To: 2, Duration: d}
		if tw.At(0) != 2 || !tw.Done(0) {
			t.Errorf("Duration %v: At(0) = %v, Done = %v", d, tw.At(0), tw.Done(0))
		}
	}
}

func TestTweenDone(t *testing.T) {
	tw := Tween{From: 0, To: 1, Duration: time.Second}
	if tw.Done(999*time.Millisecond) || !tw.Done(time.Second) || !tw.Done(2*time.Second) {
		t.Error("Done should flip exactly at Duration")
	}
}
