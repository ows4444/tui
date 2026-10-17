package layout

import (
	"strings"
	"testing"
)

func stackOf(w, h int, kids ...Node) []string {
	return strings.Split(Stack(kids...).Render(Size{W: w, H: h}), "\n")
}

func TestAnchorCorners(t *testing.T) {
	base := Fixed(Block(""), Size{W: 8, H: 4})
	for _, tt := range []struct {
		name string
		c    Corner
		dx   int
		dy   int
		want []string
	}{
		{"top left", TopLeft, 0, 0, []string{"X       ", "        ", "        ", "        "}},
		{"top right", TopRight, 0, 0, []string{"       X", "        ", "        ", "        "}},
		{"bottom left", BottomLeft, 0, 0, []string{"        ", "        ", "        ", "X       "}},
		{"bottom right", BottomRight, 0, 0, []string{"        ", "        ", "        ", "       X"}},
		{"center", Center, 0, 0, []string{"        ", "   X    ", "        ", "        "}},
		{"inward offset", TopRight, 2, 1, []string{"        ", "     X  ", "        ", "        "}},
		{"center nudged", Center, 1, -1, []string{"    X   ", "        ", "        ", "        "}},
		{"kept inside", BottomRight, 20, 20, nil},
	} {
		got := stackOf(8, 4, base, Anchor(tt.c, tt.dx, tt.dy, Block("X")))
		if tt.want != nil && strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
			t.Errorf("%s:\n%s\nwant\n%s", tt.name, strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
		}
		if tt.want == nil && got[0][0] != 'X' {
			t.Errorf("an offset past the edge should clamp to the Stack: %q", got)
		}
	}
}

func TestAnchorFollowsTheStackSizeAndStacksOnLayer(t *testing.T) {
	n := Anchor(BottomRight, 0, 0, Block("ab"))
	for _, s := range []Size{{W: 5, H: 2}, {W: 9, H: 3}} {
		got := stackOf(s.W, s.H, Fixed(Block(""), s), n)
		last := got[len(got)-1]
		if !strings.HasSuffix(last, "ab") || len(last) != s.W {
			t.Errorf("size %v: %q", s, got)
		}
	}
	over := stackOf(6, 1, Layer(1, Anchor(TopLeft, 0, 0, Block("on"))), Layer(0, Anchor(TopLeft, 0, 0, Block("under"))))
	if over[0] != "onder " {
		t.Errorf("z order lost: %q", over)
	}
	if m := Stack(Anchor(Center, 0, 0, Block("abc"))).Measure(Loose(Size{W: 20, H: 10})); m != (Size{W: 3, H: 1}) {
		t.Errorf("Measure %v", m)
	}
	if got := Anchor(Center, 0, 0, Block("ab")).Render(Size{W: 4, H: 1}); got != "ab  " {
		t.Errorf("outside a Stack it is its child: %q", got)
	}
}

func TestAspect(t *testing.T) {
	n := Aspect(2, 1, Block("#"))
	if m := n.Measure(Loose(Size{W: 20, H: 4})); m != (Size{W: 8, H: 4}) {
		t.Errorf("fit 20x4 at 2:1 = %v; want 8x4", m)
	}
	if m := n.Measure(Loose(Size{W: 6, H: 10})); m != (Size{W: 6, H: 3}) {
		t.Errorf("fit 6x10 at 2:1 = %v; want 6x3", m)
	}
	if m := n.Measure(Loose(Size{W: 10, H: Unbounded})); m != (Size{W: 10, H: 5}) {
		t.Errorf("width-bound = %v", m)
	}
	if m := n.Measure(Loose(Size{W: Unbounded, H: 5})); m != (Size{W: 10, H: 5}) {
		t.Errorf("height-bound = %v", m)
	}
	if m := n.Measure(Unconstrained()); m.W != 2*m.H {
		t.Errorf("natural size %v is not 2:1", m)
	}
	out := strings.Split(Aspect(2, 1, Fixed(Block(""), Size{W: 1, H: 1})).Render(Size{W: 10, H: 3}), "\n")
	if len(out) != 3 {
		t.Fatalf("render rows %d", len(out))
	}
	if Aspect(0, -1, Block("x")).(aspectNode).w != 1 {
		t.Error("a ratio under 1 is not 1")
	}
	if Aspect(2, 1, Block("x")).Render(Size{}) != "" {
		t.Error("an empty size must render nothing")
	}
}
