package hovercard_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/hovercard"
)

// The pointer moving onto the name opens its card; Esc closes it.
func Example() {
	view := "by @ada" + strings.Repeat("\n"+strings.Repeat(" ", 24), 6)
	card := hovercard.New("Ada Lovelace", "Wrote the first\nprogram", hittest.Rect{X: 3, Y: 0, W: 4, H: 1})
	card.Mouse, card.Bounds = true, hittest.Rect{W: 24, H: 7}
	card, _ = card.Update(tui.MouseEvent{X: 4, Y: 0, Action: tui.MouseActionMotion})
	// Rows are trimmed of the view's blank cells for the comparison.
	for _, row := range strings.Split(ansi.StripANSI(card.Render(view)), "\n") {
		fmt.Println(strings.TrimRight(row, " "))
	}
	card, _ = card.Update(tui.Key{Type: tui.KeyEsc})
	fmt.Println(card.Open())
	// Output:
	// by @ada
	//    ┌─────────────────┐
	//    │ Ada Lovelace    │
	//    │                 │
	//    │ Wrote the first │
	//    │ program         │
	//    └─────────────────┘
	// false
}
