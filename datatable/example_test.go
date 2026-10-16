package datatable_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/datatable"
)

// A datatable draws a header and a window of rows, and tracks the cursor.
func Example() {
	m := datatable.New([]string{"name", "qty"}, [][]string{{"apple", "3"}, {"pear", "5"}, {"plum", "8"}})
	m.Height = 3
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	for _, line := range strings.Split(ansi.StripANSI(m.View()), "\n") {
		fmt.Println(strings.TrimRight(line, " "))
	}
	fmt.Println("cursor row:", m.Cursor())
	// Output:
	// name   qty
	//   ──────────
	//   apple  3
	// > pear   5
	//   plum   8
	// cursor row: 1
}

// Sort by a column from code, then narrow the rows with the filter: "/" opens
// the filter line, typed text keeps the rows that fuzzy-match, and SourceRow
// says which of the rows you passed in a shown row is.
func ExampleModel_filter() {
	m := datatable.New(
		[]string{"Name", "City"},
		[][]string{{"Carol", "Lahore"}, {"alice", "Tokyo"}, {"Bob", "Lahore"}},
	)
	m.Filter = true
	m.SortBy(0, false) // Bob, Carol, alice: capitals first, as bytes compare

	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "/"})
	for _, r := range "lah" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	for i, row := range m.Rows {
		fmt.Println(row[0], "was row", m.SourceRow(i))
	}
	// Output:
	// Bob was row 2
	// Carol was row 0
}

// SetCell, InsertRow and RemoveRow edit the source rows and keep a sort or
// filter in force; they never touch the slices the caller passed in.
func ExampleModel_SetCell() {
	m := datatable.New([]string{"name", "qty"}, [][]string{{"apple", "3"}, {"pear", "5"}})
	m.SetCell(1, 1, "6")
	m.InsertRow(0, []string{"fig", "1"})
	m.RemoveRow(2)
	fmt.Println(m.Rows)
	// Output: [[fig 1] [apple 3]]
}
