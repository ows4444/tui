package textinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeMeasureAndExactRender(t *testing.T) {
	m := New()
	m.Prompt = "Name: "
	m.Placeholder = "Ada"
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 6 + 3 + 1, H: 1}) {
		t.Errorf("placeholder Measure = %v", got)
	}
	m.SetValue("Grace Hopper")
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 6 + 12 + 1, H: 1}) {
		t.Errorf("value Measure = %v", got)
	}
	for _, s := range []layout.Size{{W: 30, H: 1}, {W: 12, H: 3}, {W: 3, H: 1}, {W: 7, H: 1}} {
		lines := strings.Split(m.LayoutNode().Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Errorf("%v: row width %d (%q)", s, ansi.Width(l), l)
			}
		}
	}
}

func TestLongValueScrollsAroundTheCursor(t *testing.T) {
	m := New()
	m.Prompt = "> "
	m.SetValue("0123456789abcdefghij") // cursor at the end
	m.Focus()
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 10, H: 1}))
	if !strings.HasPrefix(out, "> ") || !strings.Contains(out, "ij") {
		t.Errorf("cursor end should stay in view: %q", out)
	}
	m.SetCursor(0)
	out = ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 10, H: 1}))
	if !strings.Contains(out, "0123") {
		t.Errorf("cursor at start should show the start: %q", out)
	}
	if m.Width != 0 {
		t.Error("rendering changed the Model's Width")
	}
}
