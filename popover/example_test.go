package popover_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/popover"
)

// A popover draws a small box anchored at a cell over the base screen.
func Example() {
	base := strings.TrimRight(strings.Repeat(strings.Repeat(".", 14)+"\n", 5), "\n")
	m := popover.New("tip", 2, 1)
	m.Show()
	fmt.Println(strings.Contains(ansi.StripANSI(m.Render(base)), "tip"))
	// Output:
	// true
}
