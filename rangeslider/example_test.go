package rangeslider_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/rangeslider"
)

// Two thumbs on one track. The arrows move the active thumb, Space switches to
// the other, and neither passes the other.
func Example() {
	price := rangeslider.New(0, 100)
	price.Width = 11
	price.ShowValue = true
	price.SetRange(20, 80)
	price.Focus()
	price, _ = price.Update(tui.Key{Type: tui.KeyPgUp}) // the low thumb up by ten
	price, _ = price.Update(tui.Key{Type: tui.KeySpace})
	price, _ = price.Update(tui.Key{Type: tui.KeyPgDown}) // the high thumb down by ten
	fmt.Println(ansi.StripANSI(price.View()))
	// Output:
	// <░░░●███●░░░> 30 to 70
}
