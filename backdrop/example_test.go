package backdrop_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/backdrop"
	"github.com/ows4444/tui/dialog"
)

// The frame is dimmed first and the dialog drawn over the result.
func Example() {
	view := "Inbox  3 unread         \n                        \n                        \n                        \n                        "
	dim := backdrop.New()
	ask := dialog.New("Delete?", "")
	frame := ask.Render(dim.Render(view))
	fmt.Println(ansi.StripANSI(frame) == ansi.StripANSI(ask.Render(view)))
	fmt.Println(frame == ask.Render(view))
	// Output:
	// true
	// false
}
