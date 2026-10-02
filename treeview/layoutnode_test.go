package treeview

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func bigTree() Model {
	var kids []Node
	for i := 0; i < 20; i++ {
		kids = append(kids, Node{Label: fmt.Sprintf("leaf%02d", i)})
	}
	m := New(Node{Label: "root", Children: kids})
	m.toggle("0")
	return m
}

func TestLayoutNodeMeasureAndExactRender(t *testing.T) {
	m := bigTree()
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H != 21 || got.W < 8 {
		t.Errorf("Measure = %v", got)
	}
	if got := New().LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
	for _, s := range []layout.Size{{W: 20, H: 5}, {W: 4, H: 3}, {W: 30, H: 40}, {W: 1, H: 1}} {
		lines := strings.Split(m.LayoutNode().Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Errorf("%v: width %d", s, ansi.Width(l))
			}
		}
	}
}

func TestCursorRowStaysVisible(t *testing.T) {
	m := bigTree()
	for _, cursor := range []int{0, 9, 20} {
		m.SetCursor(cursor)
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 4}))
		want := "leaf" + fmt.Sprintf("%02d", cursor-1)
		if cursor == 0 {
			want = "root"
		}
		if !strings.Contains(out, "> ") || !strings.Contains(out, want) {
			t.Errorf("cursor %d: want the marked row %q visible in\n%s", cursor, want, out)
		}
	}
}
