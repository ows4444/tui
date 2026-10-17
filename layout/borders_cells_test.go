package layout_test

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	. "github.com/ows4444/tui/layout"
)

// DrawCells and Render must show the same screen for per-side colours.
func TestBorderColorsDrawCellsMatchRender(t *testing.T) {
	b := NewBox().Border(RoundedBorder()).Padding(0, 1, 0, 1).Title("t", AlignStart).BorderColors(ansi.Red, ansi.Green, ansi.Yellow, ansi.Blue)
	sameScreen(t, "per-side colours", BoxNode(b, Block("hello\nworld")), Loose(Size{W: 30, H: 10}))
	sameScreen(t, "per-side, one side only", BoxNode(NewBox().Border(DashedBorder()).BorderColors(nil, ansi.Red, nil, nil), Block("x")), Loose(Size{W: 30, H: 10}))
}

func TestShadowNodeSizeAndCells(t *testing.T) {
	box := NewBox().Border(NormalBorder()).Padding(0, 1, 0, 1).ShadowGlyph("#", nil)
	n := BoxNode(box, Block("hi"))
	if got := n.Measure(Loose(Size{W: 40, H: 10})); got != (Size{W: 4 + 2 + 1, H: 3 + 1}) {
		t.Fatalf("Measure = %v; want 7x4 (box 6x3 plus the shadow)", got)
	}
	sameScreen(t, "shadow", n, Loose(Size{W: 40, H: 10}))
	sameScreen(t, "shadow, coloured", BoxNode(NewBox().Border(RoundedBorder()).Shadow(ansi.BrightBlack), Block("a\nb")), Loose(Size{W: 40, H: 10}))
}

func TestAnchorAndAspectDrawLikeTheyRender(t *testing.T) {
	panel := BoxNode(NewBox().Border(NormalBorder()), Block("body\nbody\nbody"))
	stack := Stack(
		Layer(0, panel),
		Layer(1, Anchor(TopRight, 1, 0, Block("[x]"))),
		Anchor(BottomLeft, 2, 0, Block("status")),
	)
	sameScreen(t, "anchors", stack, Loose(Size{W: 30, H: 8}))
	sameScreen(t, "aspect", Aspect(2, 1, BoxNode(NewBox().Border(RoundedBorder()), Block("sq"))), Loose(Size{W: 20, H: 6}))
}
