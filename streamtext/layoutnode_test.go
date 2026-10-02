package streamtext

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

func TestLayoutNodeWrapsToTheAllottedWidth(t *testing.T) {
	m := New()
	m.SetText("the quick brown fox jumps over the lazy dog")
	m.Skip() // reveal everything
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 12, H: 6}))
	for _, l := range strings.Split(out, "\n") {
		if len(strings.TrimRight(l, " ")) > 12 {
			t.Errorf("line wider than the slot: %q", l)
		}
	}
	if !strings.Contains(strings.Join(strings.Fields(out), " "), "the quick brown fox jumps over the lazy dog") {
		t.Errorf("text lost while wrapping:\n%s", out)
	}
	if m.Width != 0 {
		t.Error("rendering changed the Model's Width")
	}
}

func TestLayoutNodeKeepsTheNewestLinesWhenTooTall(t *testing.T) {
	m := New()
	m.SetText("one two three four five six seven eight nine ten")
	m.Skip()
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 8, H: 2}))
	if !strings.Contains(out, "ten") || strings.Contains(out, "one") {
		t.Errorf("a streaming text should show its newest lines, not the oldest:\n%s", out)
	}
	if got := New().LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
}
