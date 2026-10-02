package splitpane_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/splitpane"
)

// Two panes in 21 columns, the divider moved one cell by the keyboard.
func Example() {
	m := splitpane.New(layout.Block("files"), layout.Block("preview"))
	m.SetTotal(21)
	m.Min1, m.Min2 = 4, 4
	m, _ = m.Update(tui.Key{Type: tui.KeyRight, Mod: input.ModCtrl})
	first, second := m.Sizes()
	fmt.Println(first, second)
	// Output: 11 9
}
