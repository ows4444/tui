package layout

import "testing"

// namedClickTree is a 8x3 Row: an unnamed 4-wide fill at x 0..3, then a
// Named "outer" 4x3 column at x 4..7 whose top row is Named "inner".
func namedClickTree() (Node, Size) {
	root := Row(0,
		FlexChild{Node: fill{'l', 4, 3}},
		FlexChild{Node: Named("outer", Column(0,
			FlexChild{Node: Named("inner", fill{'i', 4, 1})},
			FlexChild{Node: fill{'o', 4, 2}},
		))},
	)
	return root, Size{W: 8, H: 3}
}

// TestNamedAtDeliversNameAndRelativePosition proves criterion #95.
func TestNamedAtDeliversNameAndRelativePosition(t *testing.T) {
	root, s := namedClickTree()
	name, x, y, ok := NamedAt(root, s, 6, 2)
	if !ok || name != "outer" || x != 2 || y != 2 {
		t.Errorf("NamedAt(6,2) = %q, %d, %d, %v; want outer, 2, 2, true", name, x, y, ok)
	}
}

// TestNamedAtInnermost proves criterion #96.
func TestNamedAtInnermost(t *testing.T) {
	root, s := namedClickTree()
	name, x, y, ok := NamedAt(root, s, 5, 0)
	if !ok || name != "inner" || x != 1 || y != 0 {
		t.Errorf("NamedAt(5,0) = %q, %d, %d, %v; want inner, 1, 0, true", name, x, y, ok)
	}
}

// TestNamedAtOutsideAll proves criterion #97, including points outside the
// root altogether.
func TestNamedAtOutsideAll(t *testing.T) {
	root, s := namedClickTree()
	for _, p := range [][2]int{{1, 1}, {-1, 0}, {8, 0}, {5, 3}} {
		if name, _, _, ok := NamedAt(root, s, p[0], p[1]); ok || name != "" {
			t.Errorf("NamedAt(%d,%d) = %q, %v; want no named click", p[0], p[1], name, ok)
		}
	}
}
