package motion_test

import (
	"testing"
	"time"

	"github.com/ows4444/tui/motion"
)

func TestTimelineParallelAndReduced(t *testing.T) {
	var a, b float64
	build := func(p motion.Preference) *motion.Timeline {
		tl := &motion.Timeline{Motion: p}
		return tl.Then(motion.Tween{To: 10, Duration: time.Second}, func(v float64) { a = v }).
			With(motion.Tween{To: 4, Duration: 500 * time.Millisecond}, func(v float64) { b = v })
	}
	t0 := time.Unix(0, 0)
	tl := build(motion.Normal)
	tl.Start(t0)
	tl.Update(t0.Add(500 * time.Millisecond))
	if a != 5 || b != 4 {
		t.Errorf("parallel at 500ms: a=%v b=%v, want 5 and 4", a, b)
	}
	a, b = 0, 0
	tl = build(motion.Reduced)
	tl.Start(t0)
	if !tl.Update(t0) || a != 10 || b != 4 {
		t.Errorf("reduced: a=%v b=%v, want final values at once", a, b)
	}
}

func TestKeyframes(t *testing.T) {
	k := motion.Keyframes{{At: 0, Value: 0}, {At: 0.5, Value: 10}, {At: 1, Value: 0}}
	for _, c := range []struct{ t, want float64 }{{-1, 0}, {0, 0}, {0.25, 5}, {0.5, 10}, {0.75, 5}, {1, 0}, {2, 0}} {
		if got := k.At(c.t); got != c.want {
			t.Errorf("At(%v) = %v, want %v", c.t, got, c.want)
		}
	}
	if (motion.Keyframes{}).At(0.5) != 0 {
		t.Error("empty Keyframes should yield 0")
	}
	c := motion.ColorKeyframes{{At: 0, Color: motion.RGB{R: 0}}, {At: 1, Color: motion.RGB{R: 200, G: 100}}}
	if got := c.At(0.5); got != (motion.RGB{R: 100, G: 50}) {
		t.Errorf("color At(0.5) = %+v", got)
	}
}
