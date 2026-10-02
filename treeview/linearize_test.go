package treeview

import "testing"

func TestLinearize(t *testing.T) {
	m := New(
		Node{Label: "src", Children: []Node{{Label: "a.go"}, {Label: "b.go"}}},
		Node{Label: "README", Children: nil},
	)
	m.toggle("0")
	m.SetCursor(2)
	want := "src, level 1, expanded, item 1 of 2\n" +
		"a.go, level 2, item 1 of 2\n" +
		"b.go, level 2, item 2 of 2, selected\n" +
		"README, level 1, item 2 of 2"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	collapsed := New(Node{Label: "d", Children: []Node{{Label: "x"}}})
	if got := collapsed.Linearize(); got != "d, level 1, collapsed, item 1 of 1, selected" {
		t.Errorf("collapsed = %q", got)
	}
}
