package accordion

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

func TestLayoutNodeKeepsTheCursorSectionVisible(t *testing.T) {
	var secs []Section
	for i := 0; i < 12; i++ {
		secs = append(secs, Section{Title: fmt.Sprintf("Section %02d", i), Content: "line a\nline b\nline c"})
	}
	m := New(secs...)
	m.Toggle(2)
	m.Toggle(9)
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H != 12+3+3 {
		t.Errorf("Measure = %v, want 18 rows (12 titles + two expanded sections of 3 lines)", got)
	}
	for _, cursor := range []int{0, 2, 9, 11} {
		m.cursor = cursor
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 24, H: 5}))
		exact(t, out, 24, 5)
		if !strings.Contains(out, "> ") || !strings.Contains(out, fmt.Sprintf("Section %02d", cursor)) {
			t.Errorf("cursor %d: its title is not visible:\n%s", cursor, out)
		}
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	exact(t, New().LayoutNode().Render(layout.Size{W: 5, H: 2}), 5, 2)
}
