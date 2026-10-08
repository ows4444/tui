package toggle_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/toggle"
)

// Right turns a focused switch on.
func Example() {
	overwrite := toggle.New("Overwrite")
	overwrite.Focus()
	overwrite, _ = overwrite.Update(tui.Key{Type: tui.KeyRight})
	fmt.Println(ansi.StripANSI(overwrite.View()))
	fmt.Println(overwrite.On())
	// Output:
	// <  ●> Overwrite
	// true
}
