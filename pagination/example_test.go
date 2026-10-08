package pagination_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/pagination"
)

// Right goes to the next page; far pages fold into an ellipsis.
func Example() {
	pages := pagination.New(20)
	pages.SetPage(9)
	pages.Focus()
	pages, _ = pages.Update(tui.Key{Type: tui.KeyRight})
	// The row has a blank cell at each end, trimmed here for the comparison.
	fmt.Println(strings.TrimSpace(ansi.StripANSI(pages.View())))
	fmt.Println(pages.Page())
	// Output:
	// 1  …  9 <10> 11  …  20
	// 10
}
