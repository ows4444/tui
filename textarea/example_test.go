package textarea_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textarea"
)

// A textarea edits several lines; Enter inserts a newline.
func Example() {
	m := textarea.New()
	m.Focus()
	for _, k := range []tui.Key{
		{Type: tui.KeyRunes, Text: "hi"},
		{Type: tui.KeyEnter},
		{Type: tui.KeyRunes, Text: "there"},
	} {
		m, _ = m.Update(k)
	}
	fmt.Printf("%q\n", m.Value())
	// Output:
	// "hi\nthere"
}
