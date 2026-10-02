package commandpalette

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

func TestLayoutNodeKeepsInputAndHighlightVisible(t *testing.T) {
	m := New(commandList(12)...)
	m.Input.Prompt = "Find: "
	m.Input.SetValue("cmd")
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.H < 2 {
		t.Errorf("Measure = %v, want the input line plus the dropdown", got)
	}
	for _, s := range []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 40, H: 2}, {W: 5, H: 1}, {W: 1, H: 1}, {W: 25, H: 40}} {
		lines := exact(t, m.LayoutNode().Render(s), s.W, s.H)
		if s.W >= 8 && !strings.Contains(ansi.StripANSI(lines[0]), "Find") {
			t.Errorf("%v: the input line must stay first: %q", s, lines[0])
		}
	}
	m.highlight = 9
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 3}))
	if !strings.Contains(out, "> ") {
		t.Errorf("highlighted row not visible in a short window:\n%s", out)
	}
	if m.highlight != 9 || m.Input.Width != 0 {
		t.Error("rendering changed the Model")
	}
}
