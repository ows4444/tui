package textinput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textinput"
)

// A textinput.Model is a value: Update returns the edited copy.
func Example() {
	m := textinput.New()
	m.Focus()
	for _, r := range "héllo" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	fmt.Println(m.Value(), m.Cursor())
	// Output: héll 4
}
