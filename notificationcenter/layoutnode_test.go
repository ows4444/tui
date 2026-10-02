package notificationcenter

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
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

func TestLayoutNodeIsEmptyWithoutNotificationsAndWrapsWithThem(t *testing.T) {
	m := New(5, 0)
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
	m.Push(Notification{Message: "The nightly build finished and every test passed", Variant: widgets.VariantSuccess})
	m.Push(Notification{Message: "Disk almost full", Variant: widgets.VariantWarning})
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 26, H: 10}))
	exact(t, out, 26, 10)
	joined := flat(out)
	if !strings.Contains(joined, "nightly build finished") || !strings.Contains(joined, "Disk almost full") {
		t.Errorf("messages lost:\n%s", out)
	}
	if got := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 26, H: 100})); got.W > 26 || got.H < 5 {
		t.Errorf("Measure at width 26 = %v, want wrapped", got)
	}
	allSizes(t, m.LayoutNode())
}

// flat collapses a rendered box to its words: border glyphs become spaces and
// runs of whitespace one space, so a phrase can be searched for across the
// wrapped lines inside the box.
func flat(s string) string {
	s = strings.NewReplacer("│", " ", "┌", " ", "┐", " ", "└", " ", "┘", " ", "─", " ", "╭", " ", "╮", " ", "╰", " ", "╯", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
