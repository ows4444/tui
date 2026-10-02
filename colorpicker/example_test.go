package colorpicker_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/colorpicker"
)

// Move across the palette and press Enter to choose the color under the cursor.
func Example() {
	m := colorpicker.New(ansi.BasicColor(1), ansi.BasicColor(2), ansi.BasicColor(4))
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	fmt.Println(m.Cursor())
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(colorpicker.SelectedMsg).Color == ansi.BasicColor(2))
	// Output:
	// 1
	// true
}
