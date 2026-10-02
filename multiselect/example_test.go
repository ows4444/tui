package multiselect_test

import (
	"fmt"

	"github.com/ows4444/tui/multiselect"
)

// A multiselect keeps a set of chosen items; Toggle flips one by index.
func Example() {
	m := multiselect.NewStrings("go", "rust", "zig")
	m.Toggle(0)
	m.Toggle(2)
	fmt.Println(m.SelectedIndexes())
	// Output: [0 2]
}
