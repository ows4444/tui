package toolapproval

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

func TestLayoutNodeWrapsTheDescription(t *testing.T) {
	m := New("deploy", "Push the new build to every production server in all three regions", RiskHigh)
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 26, H: 10}))
	lines := exact(t, out, 26, 10)
	if len(strings.Fields(strings.Join(lines[1:4], " "))) < 8 {
		t.Errorf("description should wrap over several rows:\n%s", out)
	}
	if !strings.Contains(out, "deploy") || !strings.Contains(strings.Join(strings.Fields(out), " "), "Approve") {
		t.Errorf("tool name or options lost:\n%s", out)
	}
	if m.Description != "Push the new build to every production server in all three regions" {
		t.Error("rendering changed the Model")
	}
	allSizes(t, m.LayoutNode())
}
