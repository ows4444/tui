package confirm_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/confirm"
)

// Pressing y confirms yes whatever is highlighted; the answer arrives as a
// ConfirmedMsg from the returned Cmd.
func Example() {
	m := confirm.New("Delete the file?")
	_, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "y"})
	fmt.Printf("%+v\n", cmd())
	// Output:
	// {Yes:true}
}
