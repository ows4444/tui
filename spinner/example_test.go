package spinner_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/spinner"
)

// A spinner is idle until Start. Start returns the Cmd that schedules the
// first tick; a Program delivers each tick back to Update.
func Example() {
	m := spinner.New()
	fmt.Println(m.Running(), ansi.StripANSI(m.View()))
	_ = m.Start()
	fmt.Println(m.Running())
	fmt.Println(len(spinner.Frames()))
	// Output:
	// false ⠋
	// true
	// 10
}
