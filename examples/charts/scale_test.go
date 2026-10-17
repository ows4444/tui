package main

import (
	"math"
	"testing"

	"github.com/ows4444/tui/theme"
)

func TestNiceStep(t *testing.T) {
	for in, want := range map[float64]float64{0.9: 1, 1.1: 2, 3: 5, 7: 10, 0.034: 0.05, 12: 20, 480: 500, 0: 1, -3: 1, math.NaN(): 1, math.Inf(1): 1} {
		if got := niceStep(in); math.Abs(got-want) > 1e-9 {
			t.Errorf("niceStep(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestNiceRangeCoversTheData(t *testing.T) {
	for _, c := range [][2]float64{{-4, 29}, {0, 1}, {3.2, 3.9}, {1e6, 5e6}, {-1, 1}, {7, 7}, {5, 2}} {
		lo, hi, step := niceRange(c[0], c[1], 4)
		a, b := math.Min(c[0], c[1]), math.Max(c[0], c[1])
		if lo > a || hi < b || step <= 0 || hi <= lo {
			t.Errorf("niceRange(%v) = %v..%v step %v does not cover %v..%v", c, lo, hi, step, a, b)
		}
		if r := math.Mod(lo, step); math.Abs(r) > 1e-9 && math.Abs(r-step) > 1e-9 {
			t.Errorf("niceRange(%v): lo %v is not a multiple of step %v", c, lo, step)
		}
	}
}

func TestLabel(t *testing.T) {
	for _, c := range []struct {
		v, step float64
		want    string
	}{{0, 5, "0"}, {10, 5, "10"}, {2.5, 0.5, "2.5"}, {0.25, 0.05, "0.25"}, {-4, 2, "-4"}, {1.5e6, 5e5, "1.5e+06"}} {
		if got := label(c.v, c.step); got != c.want {
			t.Errorf("label(%v, %v) = %q, want %q", c.v, c.step, got, c.want)
		}
	}
}

func TestWindowZoomKeepsThePointUnderThePointer(t *testing.T) {
	w := full(100) // 0..99
	z := w.zoom(0.5, 0.25)
	if math.Abs(z.span()-w.span()/2) > 1e-9 {
		t.Fatalf("span = %v", z.span())
	}
	before := w.lo + 0.25*w.span()
	after := z.lo + 0.25*z.span()
	if math.Abs(before-after) > 1e-9 {
		t.Fatalf("the point under the pointer moved: %v -> %v", before, after)
	}
}

func TestWindowStaysInsideTheData(t *testing.T) {
	w := full(50)
	for i := 0; i < 40; i++ {
		w = w.zoom(0.7, 0.5)
	}
	if w.span() < w.minSpan()-1e-9 || w.lo < 0 || w.hi > 49 {
		t.Fatalf("window = %+v", w)
	}
	for i := 0; i < 60; i++ {
		w = w.zoom(1.4, 1)
	}
	if !w.isFull() {
		t.Fatalf("zooming out did not return to the full range: %+v", w)
	}
	z := full(50).zoom(0.2, 0.5)
	for _, dx := range []float64{-1000, 1000, 3, -3} {
		p := z.pan(dx)
		if p.lo < 0 || p.hi > 49 || math.Abs(p.span()-z.span()) > 1e-9 {
			t.Fatalf("pan %v: %+v from %+v", dx, p, z)
		}
	}
}

func TestSmallDataSetsStillZoom(t *testing.T) {
	w := full(12).zoom(0.01, 0.5)
	if w.span() != 3 {
		t.Fatalf("span = %v, want the 3-point floor", w.span())
	}
	one := full(1)
	if z := one.zoom(0.5, 0.5); z.lo != 0 || z.hi != 1 {
		t.Fatalf("a one-point set: %+v", z)
	}
}

func TestWindowMapsBetweenDataAndDots(t *testing.T) {
	w := window{lo: 10, hi: 20, n: 100}
	if w.col(10, 101) != 0 || w.col(20, 101) != 100 || w.col(15, 101) != 50 {
		t.Fatalf("cols = %d %d %d", w.col(10, 101), w.col(20, 101), w.col(15, 101))
	}
	for _, c := range []int{0, 25, 50, 100} {
		if x := w.at(c, 101); w.col(x, 101) != c {
			t.Fatalf("round trip of column %d gave %d", c, w.col(x, 101))
		}
	}
	if w.first() != 10 || w.last() != 20 {
		t.Fatalf("first/last = %d/%d", w.first(), w.last())
	}
	if (window{lo: 9.5, hi: 20.5, n: 100}).first() != 10 || (window{lo: 9.5, hi: 20.5, n: 100}).last() != 20 {
		t.Fatal("a window between points should include only the points inside it")
	}
	if w.nearest(14.6) != 15 || w.nearest(-5) != 0 || w.nearest(500) != 99 {
		t.Fatal("nearest")
	}
}

func TestRowMapping(t *testing.T) {
	if row(10, 0, 10, 40) != 0 || row(0, 0, 10, 40) != 39 || row(5, 0, 10, 41) != 20 {
		t.Fatalf("rows = %d %d %d", row(10, 0, 10, 40), row(0, 0, 10, 40), row(5, 0, 10, 41))
	}
	if row(99, 0, 10, 40) != 0 || row(-99, 0, 10, 40) != 39 {
		t.Fatal("values outside the range are not clamped")
	}
	if row(3, 5, 5, 40) != 39 {
		t.Fatal("an empty range should not divide by zero")
	}
}

func TestBounds(t *testing.T) {
	data := [][]float64{{1, 5, 3, math.NaN(), 9}, {-2, 0, 100, 1, 1}}
	lo, hi, ok := bounds(data, []bool{true, false}, full(5))
	if !ok || lo != 1 || hi != 9 {
		t.Fatalf("bounds = %v %v %v", lo, hi, ok)
	}
	lo, hi, _ = bounds(data, []bool{true, true}, window{lo: 1, hi: 3, n: 5})
	if lo != 0 || hi != 100 {
		t.Fatalf("windowed bounds = %v %v", lo, hi)
	}
	if _, _, ok := bounds(data, []bool{false, false}, full(5)); ok {
		t.Fatal("bounds of nothing")
	}
	if _, _, ok := bounds([][]float64{{math.NaN(), math.Inf(1)}}, []bool{true}, full(2)); ok {
		t.Fatal("NaN and Inf should be ignored")
	}
}

func TestDatasetsAreConsistent(t *testing.T) {
	for _, d := range datasets() {
		if len(d.series) < 2 || d.points() < 12 {
			t.Errorf("%s: %d series, %d points", d.name, len(d.series), d.points())
		}
		for _, s := range d.series {
			if len(s.values) != d.points() {
				t.Errorf("%s/%s has %d points, want %d", d.name, s.name, len(s.values), d.points())
			}
			for _, v := range s.values {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Errorf("%s/%s holds %v", d.name, s.name, v)
				}
			}
		}
		if d.x(0) == "" || d.x(d.points()-1) == "" {
			t.Errorf("%s has an empty x label", d.name)
		}
	}
}

func themeDark() theme.Theme { return theme.DarkTheme() }
