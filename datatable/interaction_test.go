package datatable

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
)

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func manyRows(n int) [][]string {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{"r" + string(rune('a'+i)), "1"}
	}
	return rows
}

func TestRebindDownToJ(t *testing.T) {
	m := New([]string{"A", "B"}, testRows())
	m.KeyMap.Down = keymap.NewBinding("down", "j")
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 0 {
		t.Fatalf("arrow moved after rebind: %d", m.Cursor())
	}
	m, _ = m.Update(runeKey('j'))
	if m.Cursor() != 1 {
		t.Fatalf("j did not move: %d", m.Cursor())
	}
}

func TestLiteralModelUsesDefaultKeyMap(t *testing.T) {
	m := Model{Headers: []string{"A"}, Rows: testRows()}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 1 {
		t.Fatalf("Cursor = %d, want 1", m.Cursor())
	}
}

func TestBindingsHaveDescriptions(t *testing.T) {
	bs := Model{}.Bindings()
	if len(bs) == 0 {
		t.Fatal("no bindings")
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

func TestMouseClickSelectsRow(t *testing.T) {
	m := New([]string{"A", "B"}, manyRows(6))
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 5, Y: 4, W: 20, H: 8}
	// Header at y=4, divider at y=5, row 0 at y=6, so row 3 is at y=9.
	m, _ = m.Update(tui.MouseEvent{X: 6, Y: 9, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 3 {
		t.Fatalf("Cursor = %d, want 3", m.Cursor())
	}
	m.Mouse = false
	m, _ = m.Update(tui.MouseEvent{X: 6, Y: 6, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 3 {
		t.Fatalf("mouse off still selected: %d", m.Cursor())
	}
}

func TestMouseWheelStep(t *testing.T) {
	m := New([]string{"A", "B"}, manyRows(20))
	m.Height = 5
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 7}
	wheel := func(b tui.MouseButton) tui.MouseEvent {
		return tui.MouseEvent{X: 1, Y: 1, Button: b, Action: tui.MouseActionPress}
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Offset() != 3 {
		t.Fatalf("default step: Offset = %d, want 3", m.Offset())
	}
	m.WheelStep = 2
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Offset() != 5 {
		t.Fatalf("step 2: Offset = %d, want 5", m.Offset())
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp))
	if m.Offset() != 3 {
		t.Fatalf("wheel up: Offset = %d, want 3", m.Offset())
	}
	m.WheelStep = 100
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Offset() != 15 {
		t.Fatalf("clamp: Offset = %d, want 15", m.Offset())
	}
}

func TestSortOrdersRowsAndShowsDirection(t *testing.T) {
	m := New([]string{"Name", "Age"}, [][]string{{"Bob", "30"}, {"Al", "5"}, {"Cy", "100"}})
	m, _ = m.Update(key(tui.KeyRight)) // current column: Age
	m, _ = m.Update(runeKey('s'))
	if got := []string{m.Rows[0][0], m.Rows[1][0], m.Rows[2][0]}; strings.Join(got, ",") != "Al,Bob,Cy" {
		t.Fatalf("ascending numeric order = %v", got)
	}
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "Age ^") {
		t.Fatalf("header lacks ascending marker:\n%s", v)
	}
	m, _ = m.Update(runeKey('s'))
	if got := []string{m.Rows[0][0], m.Rows[1][0], m.Rows[2][0]}; strings.Join(got, ",") != "Cy,Bob,Al" {
		t.Fatalf("descending order = %v", got)
	}
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "Age v") {
		t.Fatalf("header lacks descending marker:\n%s", v)
	}
}

func TestHorizontalScrollKeepsCurrentColumnVisible(t *testing.T) {
	m := New([]string{"AAAA", "BBBB", "CCCC"}, [][]string{{"a1", "b1", "c1"}})
	m.Width = 10
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "AAAA") || strings.Contains(v, "CCCC") {
		t.Fatalf("initial view:\n%s", v)
	}
	m, _ = m.Update(key(tui.KeyRight))
	m, _ = m.Update(key(tui.KeyRight))
	v := ansi.StripANSI(m.View())
	if !strings.Contains(v, "CCCC") || strings.Contains(v, "AAAA") {
		t.Fatalf("scrolled view:\n%s", v)
	}
	m, _ = m.Update(key(tui.KeyLeft))
	m, _ = m.Update(key(tui.KeyLeft))
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "AAAA") {
		t.Fatalf("scrolled back view:\n%s", v)
	}
}

func TestMouseClickInNamedNodeNoBounds(t *testing.T) {
	m := New([]string{"A", "B"}, manyRows(6))
	m.Mouse = true
	m.Name = "table"
	root := layout.Row(0,
		layout.FlexChild{Node: layout.Fixed(layout.Block(""), layout.Size{W: 5, H: 1})},
		layout.FlexChild{Node: layout.Named("table", m.LayoutNode()), Grow: 1})
	m = m.WithLayout(root, layout.Size{W: 40, H: 12})
	if m.Bounds.X != 5 {
		t.Fatalf("Bounds = %+v, want X 5", m.Bounds)
	}
	// Header at y=0, divider y=1, row 3 at y=5.
	m, _ = m.Update(tui.MouseEvent{X: 6, Y: 5, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 3 {
		t.Fatalf("Cursor = %d, want 3", m.Cursor())
	}
}
