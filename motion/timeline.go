package motion

import (
	"math"
	"sort"
	"time"
)

type entry struct {
	tw       Tween
	apply    func(float64)
	begin    time.Duration
	finished bool
}

func (e *entry) end() time.Duration { return e.begin + e.tw.Duration }

// Timeline sequences and overlaps Tweens (each with an apply func that
// receives its value) against a time read from a Clock
// (or a FakeClock in tests): it holds no timer, so it is deterministic.
//
//	tl.Then(a).Then(b)           // b starts when a ends
//	tl.Then(a).With(b)           // b starts when a starts
//
// A Timeline is built, then Started and Updated from the Update loop; it is
// not safe for concurrent use.
type Timeline struct {
	// Motion is the reduced-motion preference. With Reduced, the first
	// Update applies every tween's final value and reports done. Tweens
	// carry their own Motion too.
	Motion Preference

	entries []entry
	total   time.Duration
	last    time.Duration // begin of the most recently added tween
	start   time.Time
	started bool
}

// Then adds tw, whose value goes to apply, to start when everything added so far has ended.
func (tl *Timeline) Then(tw Tween, apply func(float64)) *Timeline { return tl.At(tl.total, tw, apply) }

// With adds tw (and its apply) to start together with the tween added last.
func (tl *Timeline) With(tw Tween, apply func(float64)) *Timeline { return tl.At(tl.last, tw, apply) }

// At adds tw (and its apply) to start offset after the Timeline starts.
func (tl *Timeline) At(offset time.Duration, tw Tween, apply func(float64)) *Timeline {
	if offset < 0 {
		offset = 0
	}
	if tw.Duration < 0 {
		tw.Duration = 0
	}
	e := entry{tw: tw, apply: apply, begin: offset}
	tl.entries = append(tl.entries, e)
	tl.last = offset
	if end := e.end(); end > tl.total {
		tl.total = end
	}
	return tl
}

// Duration is the Timeline's total length.
func (tl *Timeline) Duration() time.Duration { return tl.total }

// Start begins (or restarts) the Timeline at now.
func (tl *Timeline) Start(now time.Time) {
	tl.start, tl.started = now, true
	for i := range tl.entries {
		tl.entries[i].finished = false
	}
}

// Update applies every tween that is active at now, in the order added, and
// reports whether the Timeline is complete. A tween that has begun applies
// its From value on the Update that reaches its start. An Update before
// Start, or after completion, does nothing.
func (tl *Timeline) Update(now time.Time) bool {
	if !tl.started {
		return false
	}
	elapsed := now.Sub(tl.start)
	if tl.Motion.Reduced() {
		elapsed = tl.total
	}
	done := true
	for i := range tl.entries {
		e := &tl.entries[i]
		if e.finished {
			continue
		}
		if elapsed < e.begin {
			done = false
			continue
		}
		local := elapsed - e.begin
		if e.tw.Done(local) {
			e.finished = true
		} else {
			done = false
		}
		if e.apply != nil {
			e.apply(e.tw.At(local))
		}
	}
	return done
}

// Keyframe is a Value at position At in [0, 1].
type Keyframe struct {
	At    float64
	Value float64
	// Ease shapes the segment that ends at this keyframe; nil is Linear.
	Ease Ease
}

// Keyframes interpolates a value across ordered keyframes. Positions
// outside the first and last keyframe hold their values.
type Keyframes []Keyframe

// At returns the interpolated value at t. Empty Keyframes yield 0.
func (k Keyframes) At(t float64) float64 {
	if len(k) == 0 || math.IsNaN(t) {
		return 0
	}
	if !sort.SliceIsSorted(k, func(i, j int) bool { return k[i].At < k[j].At }) {
		k = append(Keyframes(nil), k...)
		sort.SliceStable(k, func(i, j int) bool { return k[i].At < k[j].At })
	}
	if t <= k[0].At {
		return k[0].Value
	}
	for i := 1; i < len(k); i++ {
		if t <= k[i].At {
			a, b := k[i-1], k[i]
			span := b.At - a.At
			if span <= 0 {
				return b.Value
			}
			ease := b.Ease
			if ease == nil {
				ease = Linear
			}
			return a.Value + (b.Value-a.Value)*ease((t-a.At)/span)
		}
	}
	return k[len(k)-1].Value
}

// RGB is a 24-bit colour.
type RGB struct{ R, G, B uint8 }

// ColorKeyframe is a Color at position At in [0, 1].
type ColorKeyframe struct {
	At    float64
	Color RGB
	// Ease shapes the segment that ends at this keyframe; nil is Linear.
	Ease Ease
}

// ColorKeyframes interpolates a colour across ordered keyframes, channel by
// channel in RGB.
type ColorKeyframes []ColorKeyframe

// At returns the interpolated colour at t. Empty ColorKeyframes yield black.
func (k ColorKeyframes) At(t float64) RGB {
	var ch [3]Keyframes
	for _, f := range k {
		for c, v := range [3]uint8{f.Color.R, f.Color.G, f.Color.B} {
			ch[c] = append(ch[c], Keyframe{At: f.At, Value: float64(v), Ease: f.Ease})
		}
	}
	q := func(c int) uint8 { return uint8(math.Round(math.Max(0, math.Min(255, ch[c].At(t))))) }
	return RGB{q(0), q(1), q(2)}
}
