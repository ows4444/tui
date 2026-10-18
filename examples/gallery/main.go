// Command gallery is a tour of the components that sit around the main
// screen: menus and the overlays that open over it (dialog, drawer, popover,
// toast), loading states (loading bar, skeleton, scroll bar, virtual list),
// fields (email, masked, autocomplete, colour picker) and a step wizard. Each
// page is a small working screen, not a picture.
//
// Keys: F1 to F4 or Ctrl+N and Ctrl+P change the page, ? shows the keys of
// the page, Ctrl+C quits. On Menus: Enter chooses, d opens a dialog, w a
// drawer, p a popover, t a toast; Esc closes an overlay. On Loading: Up and Down
// scroll the list. On Fields: Tab moves between the four fields. On Wizard: n
// goes on, b goes back. Mouse: click a page name at the top.
//
//	go run ./examples/gallery
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	if _, err := ui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseCellMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
