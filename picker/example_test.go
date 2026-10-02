package picker_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/picker"
)

// A picker is a single-choice list: move the cursor, read it back.
func Example() {
	m := picker.NewStrings("small", "medium", "large")
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	fmt.Println(m.Cursor())
	// Output: 2
}
