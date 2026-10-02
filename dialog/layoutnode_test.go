package dialog

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

func TestLayoutNodeIsEmptyWhenClosedAndABoxWhenOpen(t *testing.T) {
	m := New("Delete file", "This cannot be undone and the file will be removed from every synced device.")
	m.Hide()
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("closed Measure = %v, want empty", got)
	}
	if out := m.LayoutNode().Render(layout.Size{W: 6, H: 2}); strings.TrimSpace(strings.ReplaceAll(out, "\n", "")) != "" {
		t.Errorf("closed Render should be blank, got %q", out)
	}
	m.Show()
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 30, H: 8}))
	exact(t, out, 30, 8)
	if !strings.Contains(out, "┌") && !strings.Contains(out, "╭") && !strings.Contains(out, "+") {
		t.Errorf("open Render should draw a border:\n%s", out)
	}
	if !strings.Contains(flat(out), "Delete file") {
		t.Errorf("open Render lacks 'Delete file':\n%s", out)
	}
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got.W < 5 || got.H < 3 {
		t.Errorf("open Measure = %v", got)
	}
	allSizes(t, m.LayoutNode())
}

func TestDialogWrapsItsMessageToTheAllottedWidth(t *testing.T) {
	m := New("Confirm", "This cannot be undone and the file will be removed from every synced device.")
	m.Show()
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 28, H: 12}))
	joined := flat(out)
	if !strings.Contains(joined, "removed from every synced device") {
		t.Errorf("message lost when wrapped:\n%s", out)
	}
	if got := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 28, H: 100})); got.W > 28 || got.H < 6 {
		t.Errorf("Measure at width 28 = %v, want a wrapped box", got)
	}
}

// flat collapses a rendered box to its words: border glyphs become spaces and
// runs of whitespace one space, so a phrase can be searched for across the
// wrapped lines inside the box.
func flat(s string) string {
	s = strings.NewReplacer("│", " ", "┌", " ", "┐", " ", "└", " ", "┘", " ", "─", " ", "╭", " ", "╮", " ", "╰", " ", "╯", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
