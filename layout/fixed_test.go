package layout

import (
	"strings"
	"testing"
)

func TestFixedZeroIsTheChild(t *testing.T) {
	n := fill{'x', 3, 2}
	if got := Fixed(n, Size{}); got != Node(n) {
		t.Errorf("Fixed(n, Size{}) = %#v, want n", got)
	}
	if got := Fixed(n, Size{W: -4, H: -1}); got != Node(n) {
		t.Errorf("negative sizes should count as free, got %#v", got)
	}
}

func TestFixedMeasure(t *testing.T) {
	n := fill{'x', 3, 2}
	for _, tc := range []struct {
		name string
		size Size
		c    Constraints
		want Size
	}{
		{"both", Size{10, 4}, Unconstrained(), Size{10, 4}},
		{"width only", Size{W: 10}, Unconstrained(), Size{10, 2}},
		{"height only", Size{H: 5}, Unconstrained(), Size{3, 5}},
		{"parent caps it", Size{10, 4}, Constraints{MaxW: 6, MaxH: 3}, Size{6, 3}},
		{"parent raises it", Size{W: 2}, Constraints{MinW: 5, MaxW: 9, MaxH: 9}, Size{5, 2}},
	} {
		if got := Fixed(n, tc.size).Measure(tc.c); got != tc.want {
			t.Errorf("%s: Measure = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestFixedRenderPadsAndClips(t *testing.T) {
	n := fill{'x', 1, 1}
	// Smaller than the slot: anchored top-left, padded with blanks.
	got := Fixed(n, Size{2, 1}).Render(Size{4, 2})
	if want := "xx  \n    "; got != want {
		t.Errorf("pad: got %q, want %q", got, want)
	}
	// Larger than the slot: clipped.
	got = Fixed(n, Size{9, 9}).Render(Size{3, 2})
	if want := "xxx\nxxx"; got != want {
		t.Errorf("clip: got %q, want %q", got, want)
	}
	// A free axis takes the slot.
	got = Fixed(n, Size{W: 2}).Render(Size{4, 2})
	if want := "xx  \nxx  "; got != want {
		t.Errorf("free height: got %q, want %q", got, want)
	}
	if got := Fixed(n, Size{2, 2}).Render(Size{0, 3}); got != "" {
		t.Errorf("empty slot: got %q", got)
	}
}

func TestFixedReplacesTheRowBasisTrick(t *testing.T) {
	body := fill{'b', 1, 1}
	old := Row(0, FlexChild{Node: body, Basis: 6})
	s := Size{W: 10, H: 2}
	// The trick stretches the body over the row's height; Fixed with only a
	// width leaves the height to the slot, which is the same.
	if a, b := old.Render(s), Fixed(body, Size{W: 6}).Render(s); a != b {
		t.Errorf("Fixed width differs from the Basis trick:\n%q\n%q", a, b)
	}
}

func TestRectsSeeThroughFixed(t *testing.T) {
	inner := Named("inner", fill{'i', 1, 1})
	root := Column(0,
		FlexChild{Node: Fixed(Column(0, FlexChild{Node: Block("ab")}, FlexChild{Node: inner}), Size{W: 5, H: 3})},
	)
	s := Size{W: 8, H: 4}
	r, ok := RectOf(root, s, "inner")
	if !ok {
		t.Fatalf("inner not found in %+v", Rects(root, s))
	}
	if want := (Rect{0, 1, 5, 1}); r != want {
		t.Errorf("inner rect = %+v, want %+v", r, want)
	}
	if !strings.Contains(root.Render(s), "iiiii") {
		t.Errorf("inner not drawn at width 5:\n%s", root.Render(s))
	}
}
