package passwordinput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/passwordinput"
)

// A passwordinput shows a mask instead of what was typed, but Value still
// returns the real text.
func Example() {
	m := passwordinput.New()
	m.Focus()
	for _, r := range "hunter2" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	fmt.Println(m.Value())
	fmt.Println(ansi.StripANSI(m.View()))
	// Output:
	// hunter2
	// •••••••
}
