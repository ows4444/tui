package tooltip_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/tooltip"
)

// The pointer moving onto the target shows the hint under it.
func Example() {
	view := "[ Save ]    \n            \n            \n            "
	tip := tooltip.New("Ctrl+S", hittest.Rect{X: 0, Y: 0, W: 8, H: 1})
	tip.Mouse, tip.Bounds = true, hittest.Rect{W: 12, H: 4}
	tip, _ = tip.Update(tui.MouseEvent{X: 3, Y: 0, Action: tui.MouseActionMotion})
	// Rows are trimmed of the view's blank cells for the comparison.
	for _, row := range strings.Split(ansi.StripANSI(tip.Render(view)), "\n") {
		fmt.Println(strings.TrimRight(row, " "))
	}
	// Output:
	// [ Save ]
	// ┌────────┐
	// │ Ctrl+S │
	// └────────┘
}
