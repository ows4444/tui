package errorretry

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

func TestLayoutNodeWrapsMessageAndHint(t *testing.T) {
	m := New("The connection to the build server was lost while uploading artifacts", 3)
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 24, H: 8}))
	lines := exact(t, out, 24, 8)
	for _, l := range lines {
		if len(strings.TrimRight(l, " ")) > 24 {
			t.Errorf("line too wide: %q", l)
		}
	}
	joined := strings.Join(strings.Fields(out), " ")
	if !strings.Contains(joined, "connection to the build server was lost") || !strings.Contains(joined, "press Enter or r to retry (0/3)") {
		t.Errorf("text lost while wrapping:\n%s", out)
	}
	if got := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 24, H: 100})); got.H < 4 {
		t.Errorf("Measure at width 24 = %v, want several rows", got)
	}
	m.retryCount = 3
	if out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 6})); !strings.Contains(strings.Join(strings.Fields(out), " "), "no retries left (3/3)") {
		t.Errorf("exhausted hint = %q", out)
	}
	allSizes(t, m.LayoutNode())
}
