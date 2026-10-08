package transferlist_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/transferlist"
)

// Down moves the cursor; Enter moves the item under it to the other list.
func Example() {
	cols := transferlist.New("Name", "Size", "Modified")
	cols.LeftTitle, cols.RightTitle = "Hidden", "Shown"
	cols.Focus()
	cols, _ = cols.Update(tui.Key{Type: tui.KeyDown})
	cols, cmd := cols.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(transferlist.ChangedMsg).Moved)
	fmt.Println(cols.Left, cols.Right)
	// Output:
	// [Size]
	// [Name Modified] [Size]
}
