// Command layoutlab is a workbench for the layout package. Pick Flex, Grid,
// Wrap or Anchor, move the sliders, and the layout redraws at once in a frame
// of the size you set. Every box shows the size it was given, a list beside it
// gives the rectangle of each named box, and the Go code that builds the
// layout is one key away.
//
// Mouse: click a mode, click or drag along a slider, click the arrows by a
// value, wheel over a slider to nudge it, click a box to select its child or
// column. Keyboard: 1 to 4 or Tab change the mode, Up and Down pick a setting,
// Left and Right change it (PgUp and PgDn by five), Home puts it back, r
// resets the mode, c switches between the rectangles and the code, ? shows
// help, q or Ctrl+C quits.
//
//	go run ./examples/layoutlab
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	if _, err := ui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
