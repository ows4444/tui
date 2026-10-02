package emailinput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/emailinput"
)

// An emailinput is a textinput that rejects whitespace as typed input and can
// say whether its value looks like an address.
func Example() {
	m := emailinput.New()
	m.Focus()
	for _, r := range "a b@c.io" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	fmt.Println(m.Value(), m.Valid())
	// Output:
	// ab@c.io true
}
