package dialog_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/dialog"
)

// A new dialog is open. Enter dismisses it and reports that with a
// DismissedMsg; Show opens it again.
func Example() {
	d := dialog.New("Saved", "Your changes were saved.")
	fmt.Println(d.Open())
	d, cmd := d.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(d.Open())
	fmt.Printf("%T\n", cmd())
	d.Show()
	fmt.Println(d.Open())
	// Output:
	// true
	// false
	// dialog.DismissedMsg
	// true
}
