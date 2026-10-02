package datepicker

import (
	"strings"
	"testing"

	"fmt"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"time"
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

func allSizes(t *testing.T, n layout.Node) {
	t.Helper()
	for _, s := range sizes {
		exact(t, n.Render(s), s.W, s.H)
	}
	exact(t, n.Render(layout.Size{W: 9, H: 2}), 9, 2)
}

func TestLayoutNodeKeepsTheCursorWeekVisible(t *testing.T) {
	for _, day := range []int{1, 15, 31} {
		m := New(time.Date(2026, time.March, day, 0, 0, 0, 0, time.UTC))
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 3}))
		lines := exact(t, out, 20, 3)
		if !strings.HasPrefix(lines[0], "Su Mo Tu We Th Fr Sa") {
			t.Errorf("day %d: the weekday row must stay: %q", day, lines[0])
		}
		if !strings.Contains(out, fmt.Sprintf("%2d", day)) {
			t.Errorf("day %d: the cursor's week is not visible:\n%s", day, out)
		}
	}
	m := New(time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC))
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H != 1+5 || got.W != 20 {
		t.Errorf("Measure = %v, want the weekday row plus March 2026's five weeks (it starts on a Sunday)", got)
	}
	allSizes(t, m.LayoutNode())
}
