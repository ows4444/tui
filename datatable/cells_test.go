package datatable

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func grid() Model {
	return New([]string{"ID", "Name", "State", "Notes"}, [][]string{
		{"1", "alpha", "ok", "first row"},
		{"2", "beta", "fail", "second row"},
		{"3", "gamma", "ok", "third row"},
	})
}

func TestCellStyleStylesTheChosenCells(t *testing.T) {
	m := grid()
	red := ansi.NewStyle().Foreground(ansi.Red)
	m.CellStyle = func(row, col int, v string) ansi.Style {
		if col == 2 && v == "fail" {
			return red
		}
		return ansi.Style{}
	}
	view := m.View()
	if !strings.Contains(view, red.Render("fail")) {
		t.Errorf("the failing cell is not styled:\n%q", view)
	}
	if strings.Contains(view, red.Render("ok")) {
		t.Error("an unmatched cell was styled")
	}
	// The same through the layout node, and through DrawCells (via View).
	out := m.LayoutNode().Render(layout.Size{W: 50, H: 6})
	if !strings.Contains(out, red.Render("fail")) {
		t.Errorf("layout node lacks the style:\n%q", out)
	}
	sameScreen(t, m, 40, 6)
}

func TestCellModeMovesAlongTheRowAndReportsTheColumn(t *testing.T) {
	m := grid()
	m.CellMode = true
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	sel := cmd().(SelectedMsg)
	if sel.Row != 1 || sel.Col != 2 || sel.Cells[2] != "fail" {
		t.Errorf("SelectedMsg = %+v, want row 1 col 2 fail", sel)
	}
	// Only the cursor cell carries the cursor style; the rest of the row does not.
	cur := m.themed().ResolvedStates().Selected.Bold()
	v := m.View()
	if !strings.Contains(v, cur.Render("fail")) {
		t.Errorf("cursor cell not highlighted:\n%q", v)
	}
	if strings.Contains(v, cur.Render("beta")) {
		t.Error("the whole row is highlighted in CellMode")
	}
}

func TestRowCursorIsUnchangedWithoutCellMode(t *testing.T) {
	m := grid()
	cur := m.themed().ResolvedStates().Selected.Bold()
	if !strings.Contains(m.View(), cur.Render("alpha")) {
		t.Errorf("row cursor lost:\n%q", m.View())
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if sel := cmd().(SelectedMsg); sel.Col != 0 {
		t.Errorf("Col = %d, want 0", sel.Col)
	}
}

func TestFrozenColumnsStayWhileTheRestScroll(t *testing.T) {
	m := New(
		[]string{"ID", "Name", "State", "Notes"},
		[][]string{{"1", "alpha", "ok", "a long note here"}, {"2", "beta", "fail", "another note"}},
	)
	m.FrozenCols = 1
	m.Width = 22
	for i := 0; i < 3; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	}
	v := ansi.StripANSI(m.View())
	lines := strings.Split(v, "\n")
	if !strings.Contains(lines[0], "ID") {
		t.Errorf("frozen column scrolled away:\n%s", v)
	}
	if !strings.Contains(lines[0], "Notes") || strings.Contains(lines[0], "Name") {
		t.Errorf("the scrolling columns are wrong:\n%s", v)
	}
	if !strings.Contains(lines[2], "1") || !strings.Contains(lines[2], "a long note") {
		t.Errorf("row cells:\n%s", v)
	}
	sameScreen(t, m, 30, 6)
}

func TestFrozenColsAreClamped(t *testing.T) {
	m := grid()
	m.FrozenCols = 99
	if got := m.frozen(); got != 3 {
		t.Errorf("frozen = %d, want 3 (one column always scrolls)", got)
	}
	m.FrozenCols = -2
	if got := m.frozen(); got != 0 {
		t.Errorf("frozen = %d, want 0", got)
	}
	m.Width = 20
	_ = m.View() // no panic with every column but one frozen
}
