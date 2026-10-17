package button_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/input"
)

// A focused button reports a press on Enter.
func Example() {
	save := button.New("Save")
	save.ID = "save"
	save.Focus()
	fmt.Println(ansi.StripANSI(save.View()))

	_, cmd := save.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(button.PressedMsg).ID)
	// Output:
	// < Save >
	// save
}

// A label can name its hot key with an ampersand: "&Save" is drawn "Save" with
// S underlined, and Alt+S presses the button.
func ExampleModel_Mnemonic() {
	m := button.New("&Save")
	m.ID = "save"
	fmt.Println(string(m.Mnemonic()), ansi.StripANSI(m.View()))
	_, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "s", Mod: input.ModAlt})
	fmt.Printf("%+v\n", cmd())
	// Output:
	// s [ Save ]
	// {ID:save On:false}
}
