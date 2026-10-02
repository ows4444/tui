package motion

import (
	"math"
	"time"
)

// Ease maps animation progress t in [0, 1] to eased progress. Every Ease in
// this package returns 0 at 0 and 1 at 1, clamps t outside [0, 1], and is
// monotonic non-decreasing except OutBack, which overshoots 1 on the way.
type Ease func(t float64) float64

func clamp01(t float64) float64 {
	switch {
	case math.IsNaN(t), t < 0:
		return 0
	case t > 1:
		return 1
	}
	return t
}

// Linear is constant speed.
func Linear(t float64) float64 { return clamp01(t) }

// InQuad starts slowly and speeds up (t squared).
func InQuad(t float64) float64 { t = clamp01(t); return t * t }

// OutQuad starts fast and slows down.
func OutQuad(t float64) float64 { t = clamp01(t); return t * (2 - t) }

// InOutQuad speeds up, then slows down.
func InOutQuad(t float64) float64 {
	t = clamp01(t)
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// InCubic is InQuad with a steeper start (t cubed).
func InCubic(t float64) float64 { t = clamp01(t); return t * t * t }

// OutCubic is OutQuad with a longer, softer stop.
func OutCubic(t float64) float64 { t = clamp01(t); u := t - 1; return u*u*u + 1 }

// InOutCubic is InOutQuad with steeper acceleration.
func InOutCubic(t float64) float64 {
	t = clamp01(t)
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := 2*t - 2
	return 0.5*u*u*u + 1
}

// OutBack overshoots the target slightly, then settles on it. It is the one
// Ease that is not monotonic: it exceeds 1 before returning to 1.
func OutBack(t float64) float64 {
	t = clamp01(t)
	const c1 = 1.70158
	const c3 = c1 + 1
	u := t - 1
	return 1 + c3*u*u*u + c1*u*u
}

// Tween interpolates a value from From to To over Duration, shaped by Ease.
// Drive it from a motion.Clock: keep the start time, and on each tick call
// At(time.Since(start)). The zero Ease is Linear.
//
// A Tween honours reduced motion: with Motion set to Reduced (for example
// motion.Detect() or the Program's ReducedMotion) it skips the animation and
// always reports To, so the app shows the end state at once.
type Tween struct {
	From, To float64
	Duration time.Duration
	Ease     Ease
	Motion   Preference
}

// At returns the value elapsed after the start: From at or before 0, To at or
// after Duration (or when Duration is not positive, or under reduced motion),
// and From to To shaped by Ease in between.
func (tw Tween) At(elapsed time.Duration) float64 {
	if tw.Motion.Reduced() || tw.Duration <= 0 || elapsed >= tw.Duration {
		return tw.To
	}
	if elapsed <= 0 {
		return tw.From
	}
	ease := tw.Ease
	if ease == nil {
		ease = Linear
	}
	t := float64(elapsed) / float64(tw.Duration)
	return tw.From + (tw.To-tw.From)*ease(t)
}

// Done reports whether the tween has reached To: always under reduced motion,
// otherwise once elapsed reaches Duration.
func (tw Tween) Done(elapsed time.Duration) bool {
	return tw.Motion.Reduced() || tw.Duration <= 0 || elapsed >= tw.Duration
}
