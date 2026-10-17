package dropdown_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/dropdown"
)

// A dropdown shows the chosen option; Enter opens the list and Enter again
// chooses the highlighted one.
func Example() {
	m := dropdown.New("Plan", "Free", "Pro", "Team")
	m.ID = "plan"
	m.Focus()
	for _, k := range []tui.KeyType{tui.KeyEnter, tui.KeyDown, tui.KeyEnter} {
		var cmd tui.Cmd
		m, cmd = m.Update(tui.Key{Type: k})
		if cmd != nil {
			fmt.Printf("%+v\n", cmd())
		}
	}
	fmt.Println(ansi.StripANSI(m.View()))
	// Output:
	// {ID:plan Index:1 Value:Pro}
	// Plan: Pro    ▸
}
