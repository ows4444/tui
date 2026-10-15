package picker_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/picker"
)

// A picker is a single-choice list: move the cursor, read it back.
func Example() {
	m := picker.NewStrings("small", "medium", "large")
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	fmt.Println(m.Cursor())
	// Output: 2
}

// With Filter on, typing narrows the list to the items that fuzzy-match,
// best first, and Enter confirms the one under the cursor.
func ExampleModel_filter() {
	m := picker.NewStrings("Go", "Rust", "Zig", "TypeScript")
	m.Filter = true
	for _, r := range "rs" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	fmt.Println(m.Query(), m.Highlighted().Label)
	// Output: rs Rust
}
