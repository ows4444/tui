// Command kanban is a three-column task board: To do, Doing and Done. Move a
// card between columns by dragging it, or with the keys, add and rename cards,
// delete one and undo the delete.
//
// Mouse: click a card to select it, double-click to rename it, drag it onto
// another column to move it (the drop row follows the pointer), click "+ add"
// at the foot of a column to add a card. Keyboard: arrows or h j k l move the
// selection, Shift+Left and Shift+Right (or < and >) move the card to the
// next column, Shift+Up and Shift+Down (or K and J) reorder it, a adds a card,
// Enter or e renames, d deletes, u undoes the delete, ? shows help, q or
// Ctrl+C quits.
//
//	go run ./examples/kanban
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
