package taginput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/taginput"
)

// Typing then Enter commits a tag.
func Example() {
	m := taginput.New()
	m.Input.Focus()
	for _, word := range []string{"go", "tui"} {
		for _, r := range word {
			m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
		}
		m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	}
	fmt.Println(m.Tags)
	// Output:
	// [go tui]
}
