package datatable

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/layout"
)

// countingSource is n generated rows that remembers which rows were asked for.
type countingSource struct {
	n    int
	seen map[int]bool
}

func (s *countingSource) Rows() int { return s.n }
func (s *countingSource) Row(i int) []string {
	if s.seen == nil {
		s.seen = map[int]bool{}
	}
	s.seen[i] = true
	return []string{"row" + strconv.Itoa(i), strconv.Itoa(i * 2)}
}

func lazy(n, height int) (Model, *countingSource) {
	src := &countingSource{n: n}
	m := NewSource([]string{"Name", "Double"}, src)
	m.Height = height
	return m, src
}

func TestSourceReadsOnlyTheWindow(t *testing.T) {
	m, src := lazy(1_000_000, 5)
	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, "row0") || !strings.Contains(out, "row4") || strings.Contains(out, "row5 ") {
		t.Fatalf("view:\n%s", out)
	}
	if len(src.seen) != 5 {
		t.Fatalf("View read %d rows of a million; want 5", len(src.seen))
	}
	buf := cellbuf.New(30, 8)
	m.DrawCells(buf, buf.Bounds())
	m.LayoutNode().Render(layout.Size{W: 30, H: 7})
	_ = m.LayoutNode().Measure(layout.Loose(layout.Size{W: 30, H: 7}))
	if len(src.seen) > 12 {
		t.Fatalf("drawing read %d distinct rows; want a window's worth", len(src.seen))
	}
}

func TestSourceMovesAndSelects(t *testing.T) {
	m, src := lazy(1_000_000, 5)
	m, _ = m.Update(tui.Key{Type: tui.KeyEnd})
	if m.Cursor() != 999_999 {
		t.Fatalf("End put the cursor on %d", m.Cursor())
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "row999999") {
		t.Fatal("last row not shown after End")
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyHome})
	m, _ = m.Update(tui.Key{Type: tui.KeyPgDown})
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if m.Cursor() != 5 || cmd == nil {
		t.Fatalf("cursor %d, cmd %v", m.Cursor(), cmd)
	}
	sel := cmd().(SelectedMsg)
	if sel.Row != 5 || sel.SourceRow != 5 || sel.Cells[0] != "row5" {
		t.Fatalf("selected %+v", sel)
	}
	if len(src.seen) > 40 {
		t.Fatalf("read %d distinct rows moving around", len(src.seen))
	}
}

func TestSourceWheel(t *testing.T) {
	m, _ := lazy(100, 5)
	m.Mouse = true
	m, _ = m.Update(tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if !strings.Contains(ansi.StripANSI(m.View()), "row3") {
		t.Fatalf("wheel did not scroll:\n%s", ansi.StripANSI(m.View()))
	}
}

func TestSourceTurnsOffSortFilterAndEdit(t *testing.T) {
	m, _ := lazy(10, 5)
	m.Filter = true
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "/"})
	if m.Filtering() {
		t.Fatal("the filter line opened over a Source")
	}
	m.SortBy(0, true)
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "s"})
	if strings.Contains(ansi.StripANSI(m.View()), " v") || strings.Contains(ansi.StripANSI(m.View()), " ^") || !strings.Contains(ansi.StripANSI(m.View()), "row0") {
		t.Fatalf("a Source was sorted:\n%s", ansi.StripANSI(m.View()))
	}
	if m.SetCell(0, 0, "x") || m.RemoveRow(0) {
		t.Fatal("edited a Source")
	}
	m.InsertRow(0, []string{"x", "y"})
	if m.Source.Rows() != 10 {
		t.Fatal("InsertRow changed a Source")
	}
}

func TestSourceLinearizeIsTheWindow(t *testing.T) {
	m, _ := lazy(1_000_000, 3)
	m.SetCursor(500_000)
	lines := strings.Split(m.Linearize(), "\n")
	if len(lines) != 3 || !strings.Contains(strings.Join(lines, "\n"), "Row 500001 of 1000000: Name: row500000, Double: 1000000, selected") {
		t.Fatalf("linearize:\n%s", strings.Join(lines, "\n"))
	}
}

func TestSourceWithoutHeightIsBounded(t *testing.T) {
	m, src := lazy(1_000_000, 0)
	_ = m.View()
	if len(src.seen) != defaultSourceRows {
		t.Fatalf("read %d rows with no height; want %d", len(src.seen), defaultSourceRows)
	}
}

func TestSetSourceAndSetRows(t *testing.T) {
	m, _ := lazy(100, 5)
	m.SetCursor(90)
	m.SetSource(&countingSource{n: 10})
	if m.Cursor() != 9 {
		t.Fatalf("cursor %d after the source shrank; want 9", m.Cursor())
	}
	m.SetRows([][]string{{"a", "b"}})
	if m.Source != nil || m.rowCount() != 1 || !strings.Contains(ansi.StripANSI(m.View()), "a") {
		t.Fatal("SetRows did not switch back to in-memory rows")
	}
}

func TestSourceEmptyAndNegative(t *testing.T) {
	m, _ := lazy(0, 5)
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd != nil || m.Linearize() != "No rows" {
		t.Fatal("an empty Source misbehaves")
	}
	m, _ = lazy(-3, 5)
	if m.rowCount() != 0 {
		t.Fatal("a negative count is not 0")
	}
}
