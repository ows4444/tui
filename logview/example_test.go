package logview_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/logview"
)

// A logview keeps the newest lines in view as they are appended.
func Example() {
	m := logview.New(20, 2)
	for _, l := range []string{"one", "two", "three"} {
		m.Append(l)
	}
	fmt.Println(ansi.StripANSI(m.View()))
	// Output:
	// two
	// three
}
