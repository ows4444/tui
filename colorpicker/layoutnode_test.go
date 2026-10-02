package colorpicker

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

func TestLayoutNodeFitsTheHexInputBesideTheSwatches(t *testing.T) {
	m := New(ansi.Red, ansi.Green, ansi.Blue)
	m.HexInput.Prompt = "#"
	m.HexInput.SetValue("abcdef123456")
	m.HexInput.Focus()
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 24, H: 1}))
	exact(t, out, 24, 1)
	if !strings.Contains(out, "#") || !strings.Contains(out, "456") {
		t.Errorf("hex input should scroll to keep its cursor in view: %q", out)
	}
	if m.HexInput.Width != 0 {
		t.Error("rendering changed the Model's input width")
	}
	allSizes(t, m.LayoutNode())
}
