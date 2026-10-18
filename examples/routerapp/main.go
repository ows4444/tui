// Command routerapp is a small multi-screen app: a sidebar of sections, a
// breadcrumb trail, a back button, and a stack of screens that you push by
// opening an item and pop by going back. Customers, their orders and each order
// are three levels deep, and the breadcrumb shows the path taken, so the same
// order reached from Orders and from a customer has a different trail.
//
// Mouse: click a section, a row (click again or double-click to open it), a
// crumb to jump back to it, the Back button, or a notice to dismiss it; the
// wheel moves the selection. Keyboard: Tab moves between the sidebar and the
// screen, arrows or j k move, Enter opens, Esc or Backspace goes back, s stars a
// customer, m ships an order, ? shows help, q or Ctrl+C quits.
//
//	go run ./examples/routerapp
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
