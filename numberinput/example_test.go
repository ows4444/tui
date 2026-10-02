package numberinput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/numberinput"
)

// A numberinput accepts digits and a single leading minus sign; other runes
// are dropped.
func Example() {
	m := numberinput.New()
	m.Focus()
	for _, r := range "-1x2-3" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	fmt.Println(m.Value())
	// Output:
	// -123
}
