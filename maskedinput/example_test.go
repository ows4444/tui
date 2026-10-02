package maskedinput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/maskedinput"
)

// A maskedinput draws each character as Mask while Value keeps the real text.
func Example() {
	m := maskedinput.New()
	m.Mask = '#'
	m.Focus()
	for _, r := range "1234" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	fmt.Println(m.Value())
	fmt.Println(ansi.StripANSI(m.View()))
	// Output:
	// 1234
	// ####
}
