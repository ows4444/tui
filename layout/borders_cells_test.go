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
