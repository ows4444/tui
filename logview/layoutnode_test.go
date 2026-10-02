package logview

import (
	"fmt"
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

func TestLayoutNodeFollowsTheTailAtAnySize(t *testing.T) {
	m := New(20, 3)
	for i := 0; i < 30; i++ {
		m.Append(fmt.Sprintf("line %02d", i))
	}
	for _, s := range []layout.Size{{W: 10, H: 3}, {W: 10, H: 6}, {W: 10, H: 1}} {
		lines := exact(t, m.LayoutNode().Render(s), s.W, s.H)
		if !strings.HasPrefix(lines[s.H-1], "line 29") {
			t.Errorf("%v: a log at its tail should still show the newest line last: %q", s, lines)
		}
	}
}

func TestLayoutNodeKeepsAScrolledPosition(t *testing.T) {
	m := New(20, 3)
	for i := 0; i < 30; i++ {
		m.Append(fmt.Sprintf("line %02d", i))
	}
	m.Viewport.GotoTop()
	lines := exact(t, m.LayoutNode().Render(layout.Size{W: 10, H: 3}), 10, 3)
	if !strings.HasPrefix(lines[0], "line 00") {
		t.Errorf("a log the user scrolled to the top must stay there: %q", lines)
	}
	if !m.Viewport.AtTop() {
		t.Error("rendering moved the Model's viewport")
	}
	for _, s := range sizes {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
}
