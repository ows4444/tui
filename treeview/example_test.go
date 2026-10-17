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

// With Filter on, typing narrows the tree to rows that fuzzy-match and their
// ancestors, which open so the match shows.
func ExampleModel_filter() {
	m := treeview.New(
		treeview.Node{Label: "src", Children: []treeview.Node{{Label: "main.go"}, {Label: "util.go"}}},
		treeview.Node{Label: "docs"},
	)
	m.Filter = true
	for _, r := range "util" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	for _, row := range m.VisibleRows() {
		fmt.Println(row.Path, row.Label)
	}
	// Output:
	// 0 src
	// 0.1 util.go
}

// A node with a Load lists its children the first time it opens. Update returns
// a Cmd that runs the Load; the LoadedMsg it delivers goes back through Update,
// which opens the node.
func ExampleNode_Load() {
	m := treeview.New(treeview.Node{Label: "logs", Load: func() []treeview.Node {
		return []treeview.Node{{Label: "app.log"}, {Label: "db.log"}}
	}})
	m, cmd := m.Update(tui.Key{Type: tui.KeyRight})
	m, _ = m.Update(cmd()) // the program does this with the Cmd's message
	for _, r := range m.VisibleRows() {
		fmt.Println(r.Label)
	}
	// Output:
	// logs
	// app.log
	// db.log
}
