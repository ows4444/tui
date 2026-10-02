package picker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func names(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("item%02d", i)
	}
	return out
}

func TestLayoutNodeMeasureAndExactRender(t *testing.T) {
	m := NewStrings(names(20)...)
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 2 + 6, H: 20}) {
		t.Errorf("Measure = %v", got)
	}
	for _, s := range []layout.Size{{W: 20, H: 5}, {W: 4, H: 3}, {W: 30, H: 30}, {W: 1, H: 1}} {
		lines := strings.Split(m.LayoutNode().Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Errorf("%v: width %d", s, ansi.Width(l))
			}
		}
	}
}

func TestCursorItemStaysVisible(t *testing.T) {
	m := NewStrings(names(20)...)
	for _, cursor := range []int{0, 7, 19} {
		m.SetCursor(cursor)
		out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 12, H: 4}))
		want := fmt.Sprintf("> item%02d", cursor)
		if !strings.Contains(out, want) {
			t.Errorf("cursor %d: %q not in\n%s", cursor, want, out)
		}
	}
	if got := NewStrings().LayoutNode().Render(layout.Size{W: 5, H: 2}); strings.TrimSpace(strings.ReplaceAll(got, "\n", "")) != "" {
		t.Errorf("empty list should render blank, got %q", got)
	}
}
