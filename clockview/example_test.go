package clockview_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/clockview"
)

// A stopwatch starts at zero and runs only after Start, which returns the Cmd
// that schedules its first tick.
func Example() {
	m := clockview.New(clockview.ModeStopwatch)
	fmt.Println(m.Running(), ansi.StripANSI(m.View()))
	_ = m.Start()
	fmt.Println(m.Running())
	m.Stop()
	fmt.Println(m.Running())
	// Output:
	// false 00:00:00
	// true
	// false
}
