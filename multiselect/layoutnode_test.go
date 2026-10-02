package multiselect

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

func names(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("item%02d", i)
	}
	return out
}

func TestLayoutNodeExactSizeAndMeasure(t *testing.T) {
	m := NewStrings(names(20)...)
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 2 + 3 + 1 + 6, H: 20}) {
		t.Errorf("Measure = %v", got)
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	exact(t, NewStrings().LayoutNode().Render(layout.Size{W: 6, H: 2}), 6, 2)
}

func TestCursorItemStaysVisibleAndCheckedStateSurvives(t *testing.T) {
	m := NewStrings(names(20)...)
	m.Toggle(7)
	for _, cursor := range []int{0, 7, 19} {
		m.SetCursor(cursor)
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 4}))
		if !strings.Contains(out, fmt.Sprintf("> [ ] item%02d", cursor)) && !strings.Contains(out, fmt.Sprintf("> [x] item%02d", cursor)) {
			t.Errorf("cursor %d not visible:\n%s", cursor, out)
		}
	}
	m.SetCursor(7)
	if out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 4})); !strings.Contains(out, "[x] item07") {
		t.Errorf("checked state lost:\n%s", out)
	}
}
