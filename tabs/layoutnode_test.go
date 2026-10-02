package tabs

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeMeasureAndExactRender(t *testing.T) {
	m := New("Home", "Logs", "Settings")
	n := m.LayoutNode()
	// " Home " 6 + " Logs " 6 + " Settings " 10, two gaps.
	if got := n.Measure(layout.Unconstrained()); got != (layout.Size{W: 24, H: 1}) {
		t.Errorf("Measure = %v", got)
	}
	if got := New().LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
	for _, s := range []layout.Size{{W: 30, H: 1}, {W: 10, H: 2}, {W: 3, H: 1}, {W: 24, H: 1}} {
		lines := strings.Split(n.Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Errorf("%v: row width %d", s, ansi.Width(l))
			}
		}
	}
	if n.Render(layout.Size{W: 0, H: 1}) != "" {
		t.Error("zero width should be empty")
	}
}

func TestActiveTabStaysVisibleWhenTheBarIsTooWide(t *testing.T) {
	labels := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot"}
	for i, want := range labels {
		m := New(labels...)
		m.SetActive(i)
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 16, H: 1}))
		if len([]rune(out)) != 16 {
			t.Fatalf("width %d", len([]rune(out)))
		}
		if !strings.Contains(out, want) {
			t.Errorf("active tab %q not visible in %q", want, out)
		}
	}
	// A tab wider than the space still shows its own left edge.
	m := New("Short", "AVeryLongTabLabel")
	m.SetActive(1)
	if out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 6, H: 1})); !strings.HasPrefix(strings.TrimLeft(out, " ["), "AVery") {
		t.Errorf("oversized active tab = %q", out)
	}
}

func TestLayoutNodeDoesNotChangeTheModel(t *testing.T) {
	m := New("a", "b", "c")
	m.SetActive(2)
	m.LayoutNode().Render(layout.Size{W: 4, H: 1})
	if m.Active() != 2 {
		t.Error("rendering changed the active tab")
	}
}
