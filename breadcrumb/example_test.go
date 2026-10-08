package breadcrumb_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/breadcrumb"
)

// Left moves back along the trail; Enter chooses the place.
func Example() {
	path := breadcrumb.New("Home", "Docs", "Guide")
	path.Focus()
	path, _ = path.Update(tui.Key{Type: tui.KeyLeft})
	// The row has a blank cell at each end, trimmed here for the comparison.
	fmt.Println(strings.TrimSpace(ansi.StripANSI(path.View())))
	_, cmd := path.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(cmd().(breadcrumb.ChosenMsg).Label)
	// Output:
	// Home /<Docs>/ Guide
	// Docs
}
