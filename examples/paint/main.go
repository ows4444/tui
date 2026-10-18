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
// redoes, c clears, s saves paint.txt in the current directory, ? shows help,
// q or Ctrl+C quits.
//
//	go run ./examples/paint
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
)

func main() {
	if _, err := tui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
