package menu

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func exact(t *testing.T, out string, wd, ht int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != ht {
		t.Fatalf("%dx%d: %d rows:\n%s", wd, ht, len(lines), out)
	}
	for i, l := range lines {
		if ansi.Width(l) != wd {
			t.Fatalf("%dx%d: row %d is %d wide: %q", wd, ht, i, ansi.Width(l), l)
		}
	}
	return lines
}

var sizes = []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 5, H: 1}, {W: 40, H: 20}, {W: 1, H: 1}}

func TestLayoutNodeFollowsTheCurrentLevel(t *testing.T) {
	var kids []Item
	for i := 0; i < 15; i++ {
		kids = append(kids, Item{Label: fmt.Sprintf("child%02d", i)})
	}
	m := New([]Item{{Label: "Parent", Children: kids}, {Label: "Other"}})
	root := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 12, H: 3}))
	if !strings.Contains(root, "Parent") {
		t.Errorf("root level = %q", root)
	}
	next, _ := m.Update(keyEnter())
	m = next
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 12, H: 4}))
	lines := exact(t, out, 12, 4)
	if !strings.Contains(lines[0], "child00") {
		t.Errorf("drilled-in level = %q", lines)
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
}
