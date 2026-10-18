// Command paint draws on a canvas of coloured pixels: pen, eraser, line,
// rectangle and flood fill, eight colours, three brush sizes, undo and redo,
// and save as text.
//
// Mouse: drag with the left button to draw, the right button to erase, the
// wheel to change the brush size; click a tool or a colour in the top rows. For
// line and rectangle, drag from one corner to the other. Keyboard: arrows move
// the cursor (Shift moves five), Space draws (pen: Space starts and stops a
// stroke; line and rectangle: Space sets the corner, then Space again),
// p e l r f pick the tool, 1 to 8 the colour, [ ] the brush size, u undoes, U
// redoes, c clears, s saves the picture as text to the file named by -out (it
// writes nothing without it), ? shows help, q or Ctrl+C quits.
//
//	go run ./examples/paint
//	go run ./examples/paint -out art.txt
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	out := flag.String("out", "", "file that s saves the picture to (empty: saving is off)")
	flag.Parse()
	m := newModel()
	m.path = *out
	if _, err := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
