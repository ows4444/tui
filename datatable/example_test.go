package datatable_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/datatable"
)

// A datatable draws a header and a window of rows, and tracks the cursor.
func Example() {
	m := datatable.New([]string{"name", "qty"}, [][]string{{"apple", "3"}, {"pear", "5"}, {"plum", "8"}})
	m.Height = 3
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	for _, line := range strings.Split(ansi.StripANSI(m.View()), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	fmt.Println("cursor row:", m.Cursor())
	// Output:
	// name   qty
	//   ──────────
	//   apple  3
	// > pear   5
	//   plum   8
	// cursor row: 1
}
