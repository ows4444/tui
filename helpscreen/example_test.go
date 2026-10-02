package helpscreen_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/helpscreen"
	"github.com/ows4444/tui/keymap"
)

// A helpscreen lists a registry's bindings as an overlay on the screen.
func Example() {
	var reg keymap.Registry
	reg.Add(keymap.NewBinding("quit", "q"))
	m := helpscreen.FromRegistry(&reg, "")
	m.Show()
	base := strings.TrimRight(strings.Repeat(strings.Repeat(" ", 30)+"\n", 8), "\n")
	out := ansi.StripANSI(m.Render(base))
	fmt.Println(m.Open(), strings.Contains(out, "quit"))
	// Output:
	// true true
}
