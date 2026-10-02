package menu_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/menu"
)

// Enter on an item with Children opens the next level; Enter on a leaf
// reports it.
func Example() {
	m := menu.New([]menu.Item{
		{Label: "File", Children: []menu.Item{{Label: "Open", Value: "open"}, {Label: "Quit", Value: "quit"}}},
		{Label: "Help", Value: "help"},
	})
	enter := tui.Key{Type: tui.KeyEnter}
	m, _ = m.Update(enter) // File: opens the submenu
	_, cmd := m.Update(enter)
	fmt.Println(cmd().(menu.SelectedMsg).Item.Value)
	// Output:
	// open
}
