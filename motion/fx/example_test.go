package fx_test

import (
	"fmt"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/motion/fx"
)

// A panel sweeps in from the left: half way through, half of it shows.
func Example() {
	panel := layout.Block("abcdefgh\nijklmnop")
	p := fx.Progress(150*time.Millisecond, 300*time.Millisecond, nil, motion.Normal)
	out := ansi.StripANSI(fx.Wrap(panel, fx.Sweep{}, p).Render(layout.Size{W: 8, H: 2}))
	fmt.Printf("%q\n", out)
	// Output: "abcd    \nijkl    "
}
