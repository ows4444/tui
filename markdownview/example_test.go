package markdownview_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/markdownview"
)

// Tab reaches the link, Enter follows it. An in-page link scrolls; any other
// is reported as a LinkMsg for the program to open.
func Example() {
	doc := "# Docs\n\nSee the [site](https://example.com).\n"
	r := markdownview.New(doc, 40, 6)
	r, _ = r.Update(tui.Key{Type: tui.KeyTab})
	_, cmd := r.Update(tui.Key{Type: tui.KeyEnter})
	msg := cmd().(markdownview.LinkMsg)
	fmt.Println(msg.Text, msg.URL)
	fmt.Println(r.Headings()[0].Anchor)
	// Output:
	// site https://example.com
	// docs
}
