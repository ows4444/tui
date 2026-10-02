package faces

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

func allSizes(t *testing.T, n layout.Node) {
	t.Helper()
	for _, s := range sizes {
		exact(t, n.Render(s), s.W, s.H)
	}
	exact(t, n.Render(layout.Size{W: 9, H: 2}), 9, 2)
}

func TestLayoutNodeUsesTheLargestHeadThatFits(t *testing.T) {
	m := New()
	m.Size = Large
	lw, lh := Large.Cells()
	sw, sh := Small.Cells()
	big := m.LayoutNode().Render(layout.Size{W: lw, H: lh})
	exact(t, big, lw, lh)
	small := m.LayoutNode().Render(layout.Size{W: sw + 1, H: sh})
	exact(t, small, sw+1, sh)
	m.Size = Small
	if want := m.LayoutNode().Render(layout.Size{W: sw + 1, H: sh}); small != want {
		t.Error("a Large model in a Small-sized slot should render exactly the Small head")
	}
	if got := New().LayoutNode().Measure(layout.Unconstrained()); got.W == 0 || got.H == 0 {
		t.Errorf("Measure = %v", got)
	}
	if m.Size != Small {
		t.Error("rendering changed the Model")
	}
	allSizes(t, m.LayoutNode())
	m.ShowLabel = true
	exact(t, m.LayoutNode().Render(layout.Size{W: lw + 4, H: lh + 2}), lw+4, lh+2)
}
