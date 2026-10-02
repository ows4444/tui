package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func exactSize(t *testing.T, out string, w, h int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != h {
		t.Fatalf("%d rows, want %d:\n%s", len(lines), h, out)
	}
	for i, l := range lines {
		if ansi.Width(l) != w {
			t.Fatalf("row %d width %d, want %d: %q", i, ansi.Width(l), w, l)
		}
	}
	return lines
}

func TestLayoutNodeMeasure(t *testing.T) {
	m := New()
	m.SetValue("ab\ncdefg\nh")
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 5, H: 3}) {
		t.Errorf("Measure = %v", got)
	}
	m2 := New()
	m2.Placeholder = "type here"
	if got := m2.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 9, H: 1}) {
		t.Errorf("placeholder Measure = %v", got)
	}
}

func TestLayoutNodeShowsAWindowContainingTheCursorLine(t *testing.T) {
	m := New()
	var text []string
	for i := 0; i < 20; i++ {
		text = append(text, "line"+string(rune('A'+i)))
	}
	m.SetValue(strings.Join(text, "\n"))
	m.Focus()
	for _, cursorLine := range []int{0, 9, 19} {
		// Put the cursor at the start of that line.
		off := 0
		for i := 0; i < cursorLine; i++ {
			off += len(text[i]) + 1
		}
		m.SetCursor(off)
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 10, H: 5}))
		lines := exactSize(t, out, 10, 5)
		want := text[cursorLine]
		found := false
		for _, l := range lines {
			if strings.HasPrefix(l, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("cursor on %q: not visible in\n%s", want, out)
		}
	}
}

func TestLayoutNodeFitsWidthAndKeepsCursorInView(t *testing.T) {
	m := New()
	m.SetValue("0123456789abcdefghij") // cursor at the end
	m.Focus()
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 8, H: 1}))
	exactSize(t, out, 8, 1)
	if !strings.Contains(out, "j") {
		t.Errorf("cursor end of a long line should scroll into view: %q", out)
	}
	if m.Width != 0 {
		t.Errorf("rendering changed the Model's Width to %d", m.Width)
	}
}

func TestLayoutNodeEmptyAndDegenerate(t *testing.T) {
	m := New()
	m.Placeholder = "hello"
	exactSize(t, ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 3, H: 2})), 3, 2)
	if m.LayoutNode().Render(layout.Size{W: 5, H: 0}) != "" {
		t.Error("zero height should render empty")
	}
}
