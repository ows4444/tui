package motion

import (
	"math"
	"testing"
)

func TestEveryNamedEaseHitsItsEndsAndClamps(t *testing.T) {
	names := EaseNames()
	if len(names) != 31 {
		t.Fatalf("%d easings; want 31", len(names))
	}
	for _, n := range names {
		e, ok := ByName(n)
		if !ok {
			t.Fatalf("ByName(%q) failed", n)
		}
		if got := e(0); math.Abs(got) > 1e-9 {
			t.Errorf("%s(0) = %v", n, got)
		}
		if got := e(1); math.Abs(got-1) > 1e-9 {
			t.Errorf("%s(1) = %v", n, got)
		}
		if e(-3) != e(0) || e(7) != e(1) || math.IsNaN(e(math.NaN())) {
			t.Errorf("%s does not clamp", n)
		}
		for i := 0; i <= 100; i++ {
			if v := e(float64(i) / 100); math.IsNaN(v) || math.IsInf(v, 0) || v < -0.5 || v > 1.5 {
				t.Errorf("%s(%d%%) = %v", n, i, v)
			}
		}
	}
}

func TestMonotonicFamilies(t *testing.T) {
	for _, n := range []string{"in-quart", "out-quart", "in-out-quart", "in-quint", "out-quint", "in-out-quint",
		"in-sine", "out-sine", "in-out-sine", "in-expo", "out-expo", "in-out-expo", "in-circ", "out-circ", "in-out-circ"} {
		e, _ := ByName(n)
		prev := e(0)
		for i := 1; i <= 200; i++ {
			v := e(float64(i) / 200)
			if v < prev-1e-12 {
				t.Errorf("%s dips at %d/200: %v after %v", n, i, v, prev)
				break
			}
			prev = v
		}
	}
}

func TestKnownMidpoints(t *testing.T) {
	for _, tt := range []struct {
		name string
		at   float64
		want float64
	}{
		{"in-out-quart", 0.5, 0.5}, {"in-out-sine", 0.5, 0.5}, {"in-out-circ", 0.5, 0.5},
		{"in-out-bounce", 0.5, 0.5}, {"in-out-elastic", 0.5, 0.5}, {"in-out-expo", 0.5, 0.5},
		{"in-quart", 0.5, 0.0625}, {"out-quint", 0.5, 0.96875}, {"in-sine", 0.5, 1 - math.Sqrt2/2},
	} {
		e, _ := ByName(tt.name)
		if got := e(tt.at); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("%s(%v) = %v; want %v", tt.name, tt.at, got, tt.want)
		}
	}
}

func TestOvershooters(t *testing.T) {
	for _, n := range []string{"in-back", "in-out-back", "in-elastic", "out-elastic", "in-out-elastic"} {
		e, _ := ByName(n)
		low, high := 0.0, 1.0
		for i := 0; i <= 1000; i++ {
			v := e(float64(i) / 1000)
			low, high = math.Min(low, v), math.Max(high, v)
		}
		if low >= 0 && high <= 1 {
			t.Errorf("%s never leaves [0, 1]; it should overshoot", n)
		}
	}
	if InBounce(0.25)+OutBounce(0.75) < 0.999 || InBounce(0.25)+OutBounce(0.75) > 1.001 {
		t.Error("InBounce is not OutBounce backwards")
	}
}

func TestByNameNormalises(t *testing.T) {
	for _, n := range []string{"Out_Cubic", " out-cubic ", "OUT-CUBIC"} {
		if e, ok := ByName(n); !ok || e(0.5) != OutCubic(0.5) {
			t.Errorf("ByName(%q) did not find out-cubic", n)
		}
	}
	if _, ok := ByName("wobble"); ok {
		t.Error("an unknown name was found")
	}
}
