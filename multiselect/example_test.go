package multiselect_test

import (
	"fmt"

	"github.com/ows4444/tui"
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

// With Filter on, typing narrows the list to the items that fuzzy-match.
// Checked items stay checked while hidden.
func ExampleModel_filter() {
	m := multiselect.NewStrings("Milk", "Eggs", "Bread", "Butter")
	m.Filter = true
	for _, r := range "bu" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	m, _ = m.Update(tui.Key{Type: tui.KeySpace})
	fmt.Println(m.Query(), m.SelectedItems())
	// Output: bu [{Butter Butter}]
}
