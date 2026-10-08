package rating_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/rating"
)

// A digit sets the score.
func Example() {
	stars := rating.New(5)
	stars.ShowValue = true
	stars.Focus()
	stars, _ = stars.Update(tui.Key{Type: tui.KeyRunes, Text: "4"})
	fmt.Println(ansi.StripANSI(stars.View()))
	// Output:
	// [●●●●○] 4/5
}
