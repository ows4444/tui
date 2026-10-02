package menubar_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/menubar"
)

// F10 opens the first menu, Right moves to the next, Enter activates.
func Example() {
	m := menubar.New(
		menubar.Menu{Title: "File", Items: []menubar.Item{{Label: "Quit", Value: "quit"}}},
		menubar.Menu{Title: "Help", Items: []menubar.Item{{Label: "About", Value: "about"}}},
	)
	m, _ = m.Update(tui.Key{Type: tui.KeyF10})
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	sel := cmd().(menubar.SelectedMsg)
	fmt.Println(sel.Menu, sel.Item.Value)
	// Output:
	// 1 about
}
