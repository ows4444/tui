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

// With KillRing on, Ctrl+K, Ctrl+U and Ctrl+W save what they delete and
// Ctrl+Y puts it back.
func ExampleModel_killRing() {
	m := textinput.New()
	m.KillRing = true
	m.Focus()
	for _, r := range "one two three" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'w'}) // kills "three"
	fmt.Printf("%q\n", m.Value())
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'y'}) // yanks it back
	fmt.Printf("%q\n", m.Value())
	// Output:
	// "one two "
	// "one two three"
}
