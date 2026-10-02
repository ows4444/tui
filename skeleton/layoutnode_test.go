package skeleton

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

func TestLayoutNodeFillsTheAllottedSize(t *testing.T) {
	m := New()
	m.Width, m.Lines = 5, 1 // its own size is irrelevant inside a layout
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 5, H: 1}) {
		t.Errorf("Measure = %v", got)
	}
	if got := New().LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("zero-size Measure = %v", got)
	}
	if m.Width != 5 || m.Lines != 1 {
		t.Error("rendering changed the Model")
	}
	if strings.TrimSpace(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 3}))) == "" {
		t.Error("expected the skeleton blocks to be drawn")
	}
}
