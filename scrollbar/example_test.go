package scrollbar_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/scrollbar"
)

// A scrollbar for 100 lines with 10 showing; the down arrow scrolls one line.
func Example() {
	m := scrollbar.New(100, 10)
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	fmt.Println(m.Offset)
	fmt.Println(m.Linearize())
	// Output:
	// 1
	// Scrollbar, vertical, 1% scrolled, showing 10 of 100
}
