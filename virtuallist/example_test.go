package virtuallist_test

import (
	"fmt"

	"github.com/ows4444/tui/virtuallist"
)

// A virtual list renders only the rows in its window, so a million items
// cost the same as ten.
func Example() {
	calls := 0
	m := virtuallist.New(1_000_000, 5, func(i int) string {
		calls++
		return fmt.Sprintf("item %d", i)
	})
	_ = m.View()
	fmt.Println(calls <= 5+2*m.Overscan)
	// Output: true
}
