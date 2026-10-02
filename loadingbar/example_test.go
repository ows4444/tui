package loadingbar_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/loadingbar"
)

// An indeterminate bar of fixed width; Start returns the Cmd that animates it.
func Example() {
	m := loadingbar.New(12)
	fmt.Println(m.Running(), ansi.Width(m.View()))
	_ = m.Start()
	fmt.Println(m.Running())
	// Output:
	// false 12
	// true
}
