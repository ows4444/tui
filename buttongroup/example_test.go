package buttongroup_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/buttongroup"
)

// A single-choice group: Right moves the cursor, Enter chooses.
func Example() {
	align := buttongroup.New(buttongroup.ModeSingle, "Left", "Centre", "Right")
	align.Focus()
	align, _ = align.Update(tui.Key{Type: tui.KeyRight})
	align, _ = align.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(ansi.StripANSI(align.View()))
	fmt.Println(align.On())
	// Output:
	// [ Left ] [●Centre ] [ Right ]
	// [Centre]
}
