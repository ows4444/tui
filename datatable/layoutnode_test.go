package datatable

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func exact(t *testing.T, out string, w, h int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != h {
		t.Fatalf("%d rows, want %d:\n%s", len(lines), h, out)
	}
	for i, l := range lines {
		if ansi.Width(l) != w {
			t.Fatalf("row %d is %d wide, want %d: %q", i, ansi.Width(l), w, l)
		}
	}
	return lines
}

func bigTable(n int) Model {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{fmt.Sprintf("row%02d", i), "some-long-description-text", fmt.Sprintf("%d", i*10)}
	}
	return New([]string{"Name", "Description", "Qty"}, rows)
}

func TestLayoutNodeMeasureIsNaturalSize(t *testing.T) {
	m := bigTable(3)
	got := m.LayoutNode().Measure(layout.Unconstrained())
	// widths: 5 ("row00"), 26, 3 ("Qty"); two 2-space gaps, the 2-cell cursor gutter; header+divider+3.
	if got != (layout.Size{W: 2 + 5 + 26 + 3 + 4, H: 5}) {
		t.Errorf("Measure = %v", got)
	}
	if got := m.LayoutNode().Measure(layout.Loose(layout.Size{W: 10, H: 2})); got != (layout.Size{W: 10, H: 2}) {
		t.Errorf("constrained Measure = %v", got)
	}
	if got := New(nil, nil).LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("no headers Measure = %v", got)
	}
}

func TestLayoutNodeAtNaturalSizeMatchesView(t *testing.T) {
	m := bigTable(3)
	m.SetCursor(1)
	n := m.LayoutNode()
	s := n.Measure(layout.Unconstrained())
	got := n.Render(s)
	exact(t, got, s.W, s.H)
	if ansi.StripANSI(got) != ansi.StripANSI(m.View()) {
		t.Errorf("natural-size render differs from View:\n%s\nvs\n%s", ansi.StripANSI(got), ansi.StripANSI(m.View()))
	}
}

func TestLayoutNodeShrinksWideColumnsWithEllipsis(t *testing.T) {
	m := bigTable(2)
	out := m.LayoutNode().Render(layout.Size{W: 24, H: 4})
	lines := exact(t, ansi.StripANSI(out), 24, 4)
	if !strings.Contains(lines[2], "…") {
		t.Errorf("wide description should be cut with an ellipsis: %q", lines[2])
	}
	// The narrow columns keep their full text.
	if !strings.Contains(lines[2], "row00") || !strings.HasSuffix(strings.TrimRight(lines[2], " "), "0") {
		t.Errorf("narrow columns were altered: %q", lines[2])
	}
	if !strings.HasPrefix(lines[0], "  Name") {
		t.Errorf("header row = %q", lines[0])
	}
}

func TestLayoutNodePinsHeaderAndKeepsCursorVisible(t *testing.T) {
	m := bigTable(20)
	for _, cursor := range []int{0, 7, 19} {
		m.SetCursor(cursor)
		out := m.LayoutNode().Render(layout.Size{W: 40, H: 6}) // header + divider + 4 rows
		lines := exact(t, ansi.StripANSI(out), 40, 6)
		if !strings.HasPrefix(lines[0], "  Name") {
			t.Errorf("cursor %d: header not pinned: %q", cursor, lines[0])
		}
		want := fmt.Sprintf("row%02d", cursor)
		found := false
		for _, l := range lines[2:] {
			if strings.HasPrefix(strings.TrimLeft(l, "> "), want) {
				found = true
			}
		}
		if !found {
			t.Errorf("cursor %d: row %s not visible in\n%s", cursor, want, strings.Join(lines, "\n"))
		}
	}
	// The cursor row keeps its highlight (bold) in the windowed render.
	m.SetCursor(12)
	out := m.LayoutNode().Render(layout.Size{W: 40, H: 6})
	if !strings.Contains(out, "\x1b[") || !strings.Contains(out, "row12") {
		t.Errorf("cursor row lost its styling: %q", out)
	}
}

func TestLayoutNodeDoesNotChangeTheModelAndTinySizesAreExact(t *testing.T) {
	m := bigTable(5)
	m.SetCursor(3)
	before := m.View()
	for _, s := range []layout.Size{{W: 1, H: 1}, {W: 3, H: 2}, {W: 8, H: 3}, {W: 100, H: 30}} {
		exact(t, m.LayoutNode().Render(s), s.W, s.H)
	}
	if m.View() != before || m.Cursor() != 3 {
		t.Error("rendering the node changed the model")
	}
}

func TestLayoutNodeInsideARow(t *testing.T) {
	m := bigTable(3)
	ui := layout.Row(1,
		layout.FlexChild{Node: layout.Block("nav"), Basis: 5},
		layout.FlexChild{Node: m.LayoutNode(), Grow: 1},
	)
	exact(t, ansi.StripANSI(layout.Draw(ui, layout.Constraints{MinW: 50, MaxW: 50, MinH: 8, MaxH: 8})), 50, 8)
}
