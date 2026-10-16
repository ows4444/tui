package datatable

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func wide() Model {
	m := New([]string{"id", "name", "region", "status", "owner"}, [][]string{
		{"1", "alpha-service", "eu-west-1", "running", "ada"},
		{"2", "beta-service", "us-east-2", "stopped", "bob"},
	})
	m.FrozenCols = 1
	m.Width = 36
	return m
}

func nodeLines(m Model, w, h int) []string {
	return strings.Split(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: w, H: h})), "\n")
}

func TestLayoutNodeScrollsColumnsAndKeepsFrozenOnesInView(t *testing.T) {
	m := wide()
	lines := nodeLines(m, 36, 4)
	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "name") {
		t.Errorf("header = %q", lines[0])
	}
	if strings.Contains(lines[0], "owner") {
		t.Errorf("the last column fits at width 36?! %q", lines[0])
	}
	if strings.Contains(strings.Join(lines, "\n"), "…") {
		t.Errorf("columns were shrunk with an ellipsis, not scrolled:\n%s", strings.Join(lines, "\n"))
	}
	// Move the current column to the end: the frozen column stays, the last shows.
	for i := 0; i < 4; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	}
	lines = nodeLines(m, 36, 4)
	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "owner") {
		t.Errorf("scrolled header = %q, want id and owner", lines[0])
	}
	if strings.Contains(lines[0], "name") {
		t.Errorf("a column that scrolled off is still shown: %q", lines[0])
	}
	if !strings.Contains(lines[2], "1") || !strings.Contains(lines[2], "ada") {
		t.Errorf("row 1 = %q: the frozen id and the owner should both show", lines[2])
	}
	for i, l := range lines {
		if ansi.Width(l) > 36 {
			t.Errorf("line %d is %d wide", i, ansi.Width(l))
		}
	}
}

func TestLayoutNodeWithoutFrozenColumnsStillShrinks(t *testing.T) {
	m := wide()
	m.FrozenCols = 0
	out := strings.Join(nodeLines(m, 30, 4), "\n")
	if !strings.Contains(out, "owner") {
		t.Errorf("without frozen columns every column should still be shown, shrunk:\n%s", out)
	}
}

func TestLayoutNodeFrozenColumnsThatFitAreNotScrolled(t *testing.T) {
	m := wide()
	out := strings.Join(nodeLines(m, 120, 4), "\n")
	for _, h := range []string{"id", "name", "region", "status", "owner"} {
		if !strings.Contains(out, h) {
			t.Errorf("a wide enough node lacks %q:\n%s", h, out)
		}
	}
}
