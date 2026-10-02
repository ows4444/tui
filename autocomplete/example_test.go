package autocomplete_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/autocomplete"
)

// Typing opens a dropdown of matching suggestions; Enter accepts the
// highlighted one.
func Example() {
	m := autocomplete.New("apple", "apricot", "banana")
	m.Focus()
	for _, r := range "ap" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	fmt.Println(m.IsOpen())
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(autocomplete.AcceptedMsg).Value)
	// Output:
	// true
	// apple
}
