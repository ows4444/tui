package filepicker

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

func TestLayoutNodeWindowAroundTheCursor(t *testing.T) {
	m := Model{Dir: "/x"}
	for i := 0; i < 25; i++ {
		m.entries = append(m.entries, entryOf(fmt.Sprintf("file%02d.go", i), i%5 == 0))
	}
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H != 25 || got.W < 10 {
		t.Errorf("Measure = %v", got)
	}
	for _, cursor := range []int{0, 12, 24} {
		m.cursor = cursor
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 16, H: 4}))
		exact(t, out, 16, 4)
		if !strings.Contains(out, fmt.Sprintf("> file%02d.go", cursor)) {
			t.Errorf("cursor %d not visible:\n%s", cursor, out)
		}
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	exact(t, Model{}.LayoutNode().Render(layout.Size{W: 5, H: 2}), 5, 2)
}
