package taginput

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

func TestLayoutNodeGivesTagsTheirRoomAndScrollsTheInput(t *testing.T) {
	m := New()
	m.Input.Prompt = "+ "
	m.Tags = []string{"go", "tui"}
	m.Input.SetValue("a very long tag name that cannot fit")
	m.Input.Focus()
	for _, s := range []layout.Size{{W: 60, H: 1}, {W: 30, H: 2}, {W: 12, H: 1}, {W: 3, H: 1}, {W: 1, H: 1}} {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 1}))
	if !strings.Contains(out, "go") || !strings.Contains(out, "tui") {
		t.Errorf("tags should keep their room: %q", out)
	}
	if !strings.Contains(out, "fit") {
		t.Errorf("cursor at the end of a long value should scroll into view: %q", out)
	}
	if m.Input.Width != 0 {
		t.Error("rendering changed the Model's input Width")
	}
	if got := New().LayoutNode().Measure(layout.Unconstrained()); got.H != 1 {
		t.Errorf("Measure = %v", got)
	}
}
