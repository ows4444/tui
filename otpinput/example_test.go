package otpinput_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/otpinput"
)

// A paste fills as many cells as it has digits.
func Example() {
	code := otpinput.New(6)
	code.Group = 3
	code.Focus()
	code, _ = code.Update(tui.PasteEvent{Text: "4821"})
	fmt.Println(ansi.StripANSI(code.View()))
	fmt.Println(code.Value(), code.Complete())
	// Output:
	// [4][8][2] — [1]< >[ ]
	// 4821 false
}
