package commandpalette_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/commandpalette"
)

// Typing filters the commands; Enter selects the best match.
func Example() {
	m := commandpalette.New(
		commandpalette.Command{Name: "open file", Description: "open a file"},
		commandpalette.Command{Name: "quit", Description: "leave the app"},
	)
	m.Focus()
	for _, r := range "qu" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(commandpalette.SelectedMsg).Command.Name)
	// Output:
	// quit
}
