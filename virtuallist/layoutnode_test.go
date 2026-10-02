package virtuallist

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

func TestLayoutNodeRendersOnlyTheAllottedRows(t *testing.T) {
	calls := 0
	m := New(100000, 2, func(i int) string { calls++; return fmt.Sprintf("row %d", i) })
	m.LineDown(500)
	calls = 0
	out := m.LayoutNode().Render(layout.Size{W: 12, H: 4})
	lines := exact(t, out, 12, 4)
	if !strings.HasPrefix(lines[0], "row 500") || !strings.HasPrefix(lines[3], "row 503") {
		t.Errorf("window = %q", lines)
	}
	if calls != 4 {
		t.Errorf("RenderItem called %d times for a 4-row window, want 4 (no overscan, no full render)", calls)
	}
	if m.Height != 2 || m.Overscan == 0 {
		t.Errorf("rendering changed the Model: Height=%d Overscan=%d", m.Height, m.Overscan)
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
}

func TestLayoutNodeReclampsOffsetAtTheEnd(t *testing.T) {
	m := New(20, 5, func(i int) string { return fmt.Sprintf("row %d", i) })
	m.LineDown(15) // last window at height 5
	lines := exact(t, m.LayoutNode().Render(layout.Size{W: 8, H: 10}), 8, 10)
	if !strings.HasPrefix(lines[0], "row 10") || !strings.HasPrefix(lines[9], "row 19") {
		t.Errorf("a taller slot should re-clamp so the list ends at the bottom: %q", lines)
	}
}

func TestLayoutNodeMeasureAndEmpty(t *testing.T) {
	m := New(50, 3, func(i int) string { return "abcdefg" })
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 7, H: 50}) {
		t.Errorf("Measure = %v", got)
	}
	empty := New(0, 3, func(i int) string { return "" })
	if got := empty.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
	exact(t, empty.LayoutNode().Render(layout.Size{W: 4, H: 2}), 4, 2)
}
