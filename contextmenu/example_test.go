package contextmenu_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/contextmenu"
)

// A right-click opens the menu at the pointer; Enter on an item reports it.
func Example() {
	m := contextmenu.New(
		contextmenu.Item{Label: "Copy", Value: "copy"},
		contextmenu.Item{Separator: true},
		contextmenu.Item{Label: "Delete", Value: "delete"},
	)
	m.Mouse = true
	m, _ = m.Update(tui.MouseEvent{X: 4, Y: 2, Button: tui.MouseButtonRight, Action: tui.MouseActionPress})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(contextmenu.SelectedMsg).Item.Value)
	// Output:
	// delete
}
