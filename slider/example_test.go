package slider_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/slider"
)

// Page Up raises the value by ten steps.
func Example() {
	volume := slider.New(0, 100)
	volume.Width = 11
	volume.ShowValue = true
	volume.Focus()
	for i := 0; i < 4; i++ {
		volume, _ = volume.Update(tui.Key{Type: tui.KeyPgUp})
	}
	fmt.Println(ansi.StripANSI(volume.View()))
	// Output:
	// [████●░░░░░░] 40
}
