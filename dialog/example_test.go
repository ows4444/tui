package dialog_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
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

// MessageText takes the message as styled spans, for an app that builds its text:
// it wraps at Width with each span keeping its style.
func ExampleModel_MessageText() {
	d := dialog.New("Delete", "")
	d.Width = 22
	d.MessageText = ansi.NewText("This cannot be ", ansi.Style{}).Append("undone", ansi.NewStyle().Bold())
	fmt.Println(ansi.StripANSI(d.Linearize()))
	// Output:
	// Dialog: Delete
	// This cannot be undone
}
