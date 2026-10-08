package checkbox_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/checkbox"
)

// Space checks a focused checkbox.
func Example() {
	keep := checkbox.New("Keep metadata")
	keep.Focus()
	keep, _ = keep.Update(tui.Key{Type: tui.KeySpace})
	fmt.Println(ansi.StripANSI(keep.View()))
	fmt.Println(keep.Checked())
	// Output:
	// <x> Keep metadata
	// true
}
