package hittest_test

import (
	"fmt"

	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

// Mouse regions from a layout: label the widgets with layout.Named, ask
// layout.Rects where they are drawn, and hand the rectangles to a hittest.Map.
// No hand-computed coordinates, so the regions cannot drift from the view.
func Example_fromLayout() {
	ui := layout.Row(1,
		layout.FlexChild{Node: layout.Named("list", layout.Block("one\ntwo\nthree")), Basis: 8},
		layout.FlexChild{Node: layout.Named("detail", layout.Block("details")), Grow: 1},
	)
	size := layout.Size{W: 30, H: 5}

	var m hittest.Map[string]
	for _, p := range layout.Rects(ui, size) {
		if p.Name != "" {
			m = m.Add(p.Name, p.Rect)
		}
	}

	for _, click := range [][2]int{{2, 1}, {20, 4}} {
		if h, ok := m.At(click[0], click[1]); ok {
			fmt.Printf("click %v -> %s at %d,%d\n", click, h.ID, h.LX, h.LY)
		}
	}
	// Output:
	// click [2 1] -> list at 2,1
	// click [20 4] -> detail at 11,4
}
