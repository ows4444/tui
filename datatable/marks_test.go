package datatable

import (
	"slices"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func multi() Model {
	m := New([]string{"N"}, [][]string{{"c"}, {"a"}, {"b"}, {"d"}})
	m.Multi = true
	m.Height = 4
	return m
}

func press(m Model, k tui.Key) Model {
	m, _ = m.Update(k)
	return m
}

var space = tui.Key{Type: tui.KeySpace}

func TestSpaceMarksTheRowUnderTheCursor(t *testing.T) {
	m := multi()
	m = press(m, space)
	m = press(m, tui.Key{Type: tui.KeyDown})
	m = press(m, tui.Key{Type: tui.KeyDown})
	m = press(m, space)
	if got := m.Marked(); !slices.Equal(got, []int{0, 2}) {
		t.Fatalf("Marked = %v", got)
	}
	m = press(m, space)
	if got := m.Marked(); !slices.Equal(got, []int{0}) {
		t.Fatalf("a second Space should unmark: %v", got)
	}
}

func TestMarksShowInTheGutterAndLinearize(t *testing.T) {
	m := multi()
	m = press(m, space)
	m = press(m, tui.Key{Type: tui.KeyDown})
	m = press(m, tui.Key{Type: tui.KeyDown})
	m = press(m, space)
	lines := strings.Split(ansi.StripANSI(m.View()), "\n")
	g := strings.TrimSpace(m.themed().GlyphSet().Check)
	if !strings.HasPrefix(lines[2], " "+g) || !strings.HasPrefix(lines[4], ">"+g) || strings.HasPrefix(lines[3], " "+g) {
		t.Fatalf("gutters:\n%s", strings.Join(lines, "\n"))
	}
	if lin := m.Linearize(); !strings.Contains(lin, "Row 1 of 4: N: c, marked") || strings.Contains(lin, "Row 2 of 4: N: a, marked") {
		t.Fatalf("linearize:\n%s", lin)
	}
}

func TestMarksFollowRowsThroughSortAndFilter(t *testing.T) {
	m := multi()
	m = press(m, space) // "c", source row 0
	m.SortBy(0, false)  // a b c d: "c" is now the third
	if !slices.Equal(m.Marked(), []int{0}) {
		t.Fatalf("a sort moved the mark: %v", m.Marked())
	}
	if !m.markedRow(2) || m.markedRow(0) {
		t.Fatal("the mark did not stay with its row")
	}
}

func TestMarkAllAndClear(t *testing.T) {
	m := multi()
	m = press(m, tui.Key{Type: tui.KeyCtrl, Code: 'a'})
	if len(m.Marked()) != 4 {
		t.Fatalf("Ctrl+A marked %v", m.Marked())
	}
	m = press(m, tui.Key{Type: tui.KeyCtrl, Code: 'a'})
	if len(m.Marked()) != 0 {
		t.Fatalf("a second Ctrl+A should unmark: %v", m.Marked())
	}
	m.Mark(1, true)
	m.Mark(99, true)
	m.Mark(-1, true)
	if !slices.Equal(m.Marked(), []int{1}) || !m.IsMarked(1) {
		t.Fatalf("Mark: %v", m.Marked())
	}
	m.ClearMarks()
	if len(m.Marked()) != 0 {
		t.Fatal("ClearMarks")
	}
}

func TestMultiOffMarksNothingAndRowEditsShiftMarks(t *testing.T) {
	m := multi()
	m.Multi = false
	if m = press(m, space); len(m.Marked()) != 0 {
		t.Fatal("Space marked with Multi off")
	}
	m.Multi = true
	m.Mark(1, true)
	m.Mark(3, true)
	m.InsertRow(0, []string{"new"})
	if !slices.Equal(m.Marked(), []int{2, 4}) {
		t.Fatalf("after an insert at 0: %v", m.Marked())
	}
	m.RemoveRow(2)
	if !slices.Equal(m.Marked(), []int{3}) {
		t.Fatalf("after removing a marked row: %v", m.Marked())
	}
	m.SetRows([][]string{{"x"}, {"y"}})
	if len(m.Marked()) != 0 {
		t.Fatalf("a shorter SetRows kept stale marks: %v", m.Marked())
	}
}

func TestMarksInLayoutNodeAndBindings(t *testing.T) {
	m := multi()
	m = press(m, space)
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 6}))
	if !strings.Contains(out, ">"+strings.TrimSpace(m.themed().GlyphSet().Check)) {
		t.Fatalf("layout node lacks the mark:\n%s", out)
	}
	found := 0
	for _, b := range m.Bindings() {
		if b.Desc == "mark" || b.Desc == "mark all" {
			found++
		}
	}
	if found != 2 {
		t.Fatal("Bindings lacks the mark keys")
	}
}
