package button_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/button"
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
