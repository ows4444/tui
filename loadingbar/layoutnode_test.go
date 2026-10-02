package loadingbar

import (
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

func TestLayoutNodeDrawsTheBarAcrossTheAllottedWidth(t *testing.T) {
	m := New(10)
	lines := exact(t, m.LayoutNode().Render(layout.Size{W: 40, H: 3}), 40, 3)
	first := ansi.StripANSI(lines[0])
	if strings.Count(first, "░")+strings.Count(first, "█") != 40 {
		t.Errorf("the bar should span all 40 columns: %q", first)
	}
	if strings.TrimSpace(lines[1]) != "" {
		t.Errorf("rows after the first should be blank: %q", lines[1])
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	if m.Width != 10 {
		t.Error("rendering changed the Model's Width")
	}
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 10, H: 1}) {
		t.Errorf("Measure = %v", got)
	}
	if got := New(0).LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("zero-width Measure = %v", got)
	}
}
