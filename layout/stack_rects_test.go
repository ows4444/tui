package layout

import (
	"strings"
	"testing"
)

func TestRectCutsTileTheRectangle(t *testing.T) {
	r := Rect{X: 2, Y: 3, W: 10, H: 6}
	top, rest := r.CutTop(2)
	if top != (Rect{2, 3, 10, 2}) || rest != (Rect{2, 5, 10, 4}) {
		t.Errorf("CutTop = %v %v", top, rest)
	}
	bottom, rest := r.CutBottom(2)
	if bottom != (Rect{2, 7, 10, 2}) || rest != (Rect{2, 3, 10, 4}) {
		t.Errorf("CutBottom = %v %v", bottom, rest)
	}
	left, rest := r.CutLeft(4)
	if left != (Rect{2, 3, 4, 6}) || rest != (Rect{6, 3, 6, 6}) {
		t.Errorf("CutLeft = %v %v", left, rest)
	}
	right, rest := r.CutRight(4)
	if right != (Rect{8, 3, 4, 6}) || rest != (Rect{2, 3, 6, 6}) {
		t.Errorf("CutRight = %v %v", right, rest)
	}
	// Out-of-range sizes clamp, so the two parts always tile the whole.
	whole, none := r.CutTop(99)
	if whole != r || !none.Empty() {
		t.Errorf("CutTop(99) = %v %v", whole, none)
	}
	zero, all := r.CutLeft(-5)
	if !zero.Empty() || all != r {
		t.Errorf("CutLeft(-5) = %v %v", zero, all)
	}
	if x, y := r.Local(5, 4); x != 3 || y != 1 {
		t.Errorf("Local = %d,%d", x, y)
	}
	if !r.Contains(2, 3) || r.Contains(12, 3) || r.Contains(2, 9) {
		t.Error("Contains edges wrong")
	}
}

func TestStackDrawsInZOrderAndPlacesAbsolute(t *testing.T) {
	ui := Stack(
		Block("aaaaaa\naaaaaa\naaaaaa"),
		Layer(2, Absolute(1, 1, Block("XX"))),
		Layer(1, Absolute(2, 1, Block("YYYY"))), // below the X layer where they meet
	)
	got := Draw(ui, Constraints{MinW: 6, MaxW: 6, MinH: 3, MaxH: 3})
	lines := strings.Split(got, "\n")
	if len(lines) != 3 || lines[0] != "aaaaaa" || lines[2] != "aaaaaa" {
		t.Fatalf("Draw =\n%s", got)
	}
	// Row 1: a, then XX from the higher layer at columns 1-2, then Y over column 3-5.
	if lines[1] != "aXXYYY" {
		t.Errorf("row 1 = %q, want aXXYYY", lines[1])
	}
}

func TestStackMeasureIsTheBoundingBoxAndSkipsEmptyChildren(t *testing.T) {
	ui := Stack(Block("abc"), Absolute(4, 2, Block("zz")), Block(""))
	sz := ui.Measure(Unconstrained())
	if sz.W != 6 || sz.H != 3 {
		t.Errorf("Measure = %+v, want 6x3", sz)
	}
	if got := ui.Render(Size{W: 0, H: 3}); got != "" {
		t.Errorf("zero width render = %q", got)
	}
	// A Layer outside a Stack is its child.
	if got := Layer(5, Block("hi")).Render(Size{W: 2, H: 1}); got != "hi" {
		t.Errorf("Layer outside a Stack = %q", got)
	}
}
