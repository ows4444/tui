package radiogroup_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/radiogroup"
)

// Down moves to the next option and chooses it.
func Example() {
	size := radiogroup.New("Small", "Medium", "Large")
	size.Focus()
	size, _ = size.Update(tui.Key{Type: tui.KeyDown})
	fmt.Println(ansi.StripANSI(size.View()))
	fmt.Println(size.Value())
	// Output:
	// ( ) Small
	// <●> Medium
	// ( ) Large
	// Medium
}
