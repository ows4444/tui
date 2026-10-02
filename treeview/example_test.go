package treeview_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/treeview"
)

// Nodes start collapsed; Right expands the one under the cursor.
func Example() {
	m := treeview.New(treeview.Node{Label: "src", Children: []treeview.Node{{Label: "main.go"}, {Label: "util.go"}}})
	fmt.Println(len(m.VisibleRows()))
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	for _, r := range m.VisibleRows() {
		fmt.Println(r.Depth, r.Label)
	}
	// Output:
	// 1
	// 0 src
	// 1 main.go
	// 1 util.go
}
