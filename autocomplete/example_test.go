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

// With Fuzzy on, the typed letters need only appear in order, so "cp" offers
// "Copy Path" and "CommandPalette".
func ExampleModel_fuzzy() {
	m := autocomplete.New("CommandPalette", "Banana", "Copy Path")
	m.Fuzzy = true
	for _, r := range "cp" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(autocomplete.AcceptedMsg).Value)
	// Output: Copy Path
}
