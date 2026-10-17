package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// niceStep rounds a rough step up to 1, 2, 5 or 10 times a power of ten.
func niceStep(rough float64) float64 {
	if rough <= 0 || math.IsNaN(rough) || math.IsInf(rough, 0) {
		return 1
	}
	exp := math.Floor(math.Log10(rough))
	f := rough / math.Pow(10, exp)
	switch {
	case f <= 1:
		f = 1
	case f <= 2:
		f = 2
	case f <= 5:
		f = 5
	default:
		f = 10
	}
	return f * math.Pow(10, exp)
}

// niceRange widens [lo, hi] to round numbers about n steps apart, and returns
// the new ends and the step.
func niceRange(lo, hi float64, n int) (float64, float64, float64) {
	if hi < lo {
		lo, hi = hi, lo
	}
	if hi == lo {
		hi, lo = hi+1, lo-1
	}
	step := niceStep((hi - lo) / float64(max(1, n)))
	return math.Floor(lo/step) * step, math.Ceil(hi/step) * step, step
}

// label prints v with just enough decimals for step.
func label(v, step float64) string {
	if v == 0 {
		return "0"
	}
	decimals := 0
	if step < 1 {
		decimals = int(math.Ceil(-math.Log10(step)))
	}
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	if abs := math.Abs(v); abs >= 1e6 {
		s = fmt.Sprintf("%.3g", v)
	}
	return strings.TrimSpace(s)
}

// window is the part of the data on screen: x from lo to hi, as indexes that
// may be fractions, out of n points.
type window struct {
	lo, hi float64
	n      int
}

func full(n int) window { return window{0, float64(max(1, n-1)), n} }

// minSpan is the fewest points a zoom keeps in view.
func (w window) minSpan() float64 { return math.Min(float64(w.n-1), 3) }

func (w window) span() float64 { return w.hi - w.lo }

func (w window) isFull() bool { return w.lo <= 0 && w.hi >= float64(w.n-1) }

// clamp keeps the window inside the data and at least minSpan wide, sliding it
// rather than shrinking it where it can.
func (w window) clamp() window {
	maxX := float64(max(1, w.n-1))
	span := math.Max(w.minSpan(), math.Min(w.span(), maxX))
	if w.n <= 1 {
		return window{0, 1, w.n}
	}
	lo := math.Max(0, math.Min(w.lo, maxX-span))
	return window{lo, lo + span, w.n}
}

// zoom scales the window by factor (below 1 zooms in) keeping the data x at
// fraction f of the way across where it is.
func (w window) zoom(factor, f float64) window {
	x := w.lo + f*w.span()
	span := w.span() * factor
	return window{x - f*span, x - f*span + span, w.n}.clamp()
}

// pan slides the window by dx index units.
func (w window) pan(dx float64) window {
	return window{w.lo + dx, w.hi + dx, w.n}.clamp()
}

// first and last are the indexes of the points inside the window.
func (w window) first() int { return int(math.Ceil(w.lo - 1e-9)) }
func (w window) last() int  { return min(w.n-1, int(math.Floor(w.hi+1e-9))) }

// col maps data x to a dot column 0..dots-1.
func (w window) col(x float64, dots int) int {
	if w.span() <= 0 {
		return 0
	}
	return int(math.Round((x - w.lo) / w.span() * float64(dots-1)))
}

// at maps a dot column back to data x.
func (w window) at(col, dots int) float64 {
	if dots <= 1 {
		return w.lo
	}
	return w.lo + float64(col)/float64(dots-1)*w.span()
}

// nearest is the index of the point closest to data x.
func (w window) nearest(x float64) int {
	return max(0, min(w.n-1, int(math.Round(x))))
}

// row maps a value to a dot row 0..dots-1, 0 at the top.
func row(v, lo, hi float64, dots int) int {
	if hi <= lo {
		return dots - 1
	}
	r := int(math.Round((1 - (v-lo)/(hi-lo)) * float64(dots-1)))
	return max(0, min(dots-1, r))
}

// bounds are the smallest and largest of the values of vis series in the
// window, or ok false when there is nothing.
func bounds(data [][]float64, vis []bool, w window) (lo, hi float64, ok bool) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for s, vals := range data {
		if !vis[s] {
			continue
		}
		for i := w.first(); i <= w.last() && i < len(vals); i++ {
			if v := vals[i]; !math.IsNaN(v) && !math.IsInf(v, 0) {
				lo, hi, ok = math.Min(lo, v), math.Max(hi, v), true
			}
		}
	}
	return lo, hi, ok
}
