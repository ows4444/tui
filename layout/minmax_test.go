package layout

import "testing"

func TestMinMaxZeroIsTheChild(t *testing.T) {
	n := fill{'x', 3, 2}
	if got := MinMax(n, Size{}, Size{}); got != Node(n) {
		t.Errorf("MinMax(n, 0, 0) = %#v, want n", got)
	}
}

func TestMinMaxMeasureReportsTheBound(t *testing.T) {
	n := fill{'x', 3, 2}
	for _, tc := range []struct {
		name     string
		min, max Size
		want     Size
	}{
		{"below min", Size{5, 4}, Size{}, Size{5, 4}},
		{"above max", Size{}, Size{2, 1}, Size{2, 1}},
		{"inside bounds", Size{1, 1}, Size{9, 9}, Size{3, 2}},
		{"width only", Size{W: 6}, Size{}, Size{6, 2}},
		{"max only on height", Size{}, Size{H: 1}, Size{3, 1}},
	} {
		if got := MinMax(n, tc.min, tc.max).Measure(Unconstrained()); got != tc.want {
			t.Errorf("%s: Measure = %v, want %v", tc.name, got, tc.want)
		}
	}
}
