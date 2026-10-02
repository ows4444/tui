package splitpane

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func ctrl(t tui.KeyType) tui.Key { return tui.Key{Type: t, Mod: input.ModCtrl} }

func newModel(total int) Model {
	m := New(layout.Block("left"), layout.Block("right"))
	m.Total = total
	m.Min1, m.Min2 = 5, 7
	return m
}

func TestDefaultsToMiddleAndClamps(t *testing.T) {
	m := newModel(21)
	if m.Pos() != 10 {
		t.Fatalf("Pos %d", m.Pos())
	}
	if a, b := m.Sizes(); a != 10 || b != 10 {
		t.Fatalf("sizes %d %d", a, b)
	}
	m.SetPos(0)
	if m.Pos() != 5 {
		t.Fatalf("min1: %d", m.Pos())
	}
	m.SetPos(99)
	if a, b := m.Sizes(); a != 13 || b != 7 {
		t.Fatalf("min2: %d %d", a, b)
	}
	// Too small for both minimums: Min1 wins, no panic, sizes non-negative.
	m.SetTotal(8)
	if a, b := m.Sizes(); a != 5 || b != 2 {
		t.Fatalf("tight: %d %d", a, b)
	}
	m.SetTotal(0)
	if a, b := m.Sizes(); a != 0 || b != 0 {
		t.Fatalf("zero: %d %d", a, b)
	}
}

func TestKeys(t *testing.T) {
	m := newModel(30)
	m.SetPos(10)
	m, cmd := m.Update(ctrl(tui.KeyRight))
	if m.Pos() != 11 {
		t.Fatalf("grow %d", m.Pos())
	}
	if msg, _ := cmd().(ChangedMsg); msg.Pos != 11 {
		t.Fatalf("msg %+v", msg)
	}
	m.Step = 4
	m, _ = m.Update(ctrl(tui.KeyLeft))
	if m.Pos() != 7 {
		t.Fatalf("shrink %d", m.Pos())
	}
	m, _ = m.Update(ctrl(tui.KeyLeft))
	m, cmd = m.Update(ctrl(tui.KeyLeft))
	if m.Pos() != 5 || cmd != nil {
		t.Fatalf("clamped at min: %d, cmd %v", m.Pos(), cmd != nil)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'e'})
	if m.Pos() != 14 {
		t.Fatalf("reset %d", m.Pos())
	}
	if _, cmd = m.Update("x"); cmd != nil {
		t.Fatal("other msg must be a no-op")
	}
}

func TestRebindAndFallback(t *testing.T) {
	m := newModel(30)
	m.SetPos(10)
	m.KeyMap.Grow.Keys = []string{"]"}
	m, _ = m.Update(ctrl(tui.KeyRight))
	if m.Pos() != 10 {
		t.Fatalf("old key moved: %d", m.Pos())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "]"})
	if m.Pos() != 11 {
		t.Fatalf("rebound key: %d", m.Pos())
	}
	lit := Model{Total: 30}
	lit.SetPos(10)
	lit, _ = lit.Update(ctrl(tui.KeyRight))
	if lit.Pos() != 11 {
		t.Fatalf("literal Model: %d", lit.Pos())
	}
	for _, b := range m.Bindings() {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

func mouse(x, y int, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: a}
}

func TestMouseDragResizesWithinMinimums(t *testing.T) {
	m := newModel(0)
	m.Bounds = hittest.Rect{X: 3, Y: 2, W: 40, H: 10}
	m.SetPos(15)
	div := m.DividerRect()
	if div != (hittest.Rect{X: 18, Y: 2, W: 1, H: 10}) {
		t.Fatalf("divider %+v", div)
	}
	if m2, _ := m.Update(mouse(18, 4, tui.MouseActionPress)); m2.Dragging() {
		t.Fatal("Mouse off must ignore the mouse")
	}
	m.Mouse = true
	if m2, _ := m.Update(mouse(17, 4, tui.MouseActionPress)); m2.Dragging() {
		t.Fatal("press beside the divider started a drag")
	}
	m, _ = m.Update(mouse(18, 4, tui.MouseActionPress))
	if !m.Dragging() {
		t.Fatal("no drag on divider press")
	}
	m, cmd := m.Update(mouse(3+20, 9, tui.MouseActionMotion))
	if a, b := m.Sizes(); a != 20 || b != 19 || cmd == nil {
		t.Fatalf("drag: %d %d", a, b)
	}
	if msg, _ := cmd().(ChangedMsg); msg.Pos != 20 {
		t.Fatalf("msg %+v", msg)
	}
	m, _ = m.Update(mouse(0, 0, tui.MouseActionMotion)) // far left
	if a, b := m.Sizes(); a != 5 || b != 34 {
		t.Fatalf("left limit: %d %d", a, b)
	}
	m, _ = m.Update(mouse(500, 0, tui.MouseActionMotion)) // far right
	if a, b := m.Sizes(); a != 32 || b != 7 {
		t.Fatalf("right limit: %d %d", a, b)
	}
	m, _ = m.Update(mouse(500, 0, tui.MouseActionRelease))
	if m.Dragging() {
		t.Fatal("release did not end drag")
	}
	m, _ = m.Update(mouse(10, 0, tui.MouseActionMotion))
	if a, _ := m.Sizes(); a != 32 {
		t.Fatalf("motion after release moved to %d", a)
	}
}

func TestMouseDragRows(t *testing.T) {
	m := newModel(0)
	m.Direction = Rows
	m.Min1, m.Min2 = 2, 3
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 1, W: 30, H: 12}
	m.SetPos(4)
	m, _ = m.Update(mouse(7, 1+4, tui.MouseActionPress))
	if !m.Dragging() {
		t.Fatal("no drag")
	}
	m, _ = m.Update(mouse(7, 100, tui.MouseActionMotion))
	if a, b := m.Sizes(); a != 8 || b != 3 {
		t.Fatalf("rows: %d %d", a, b)
	}
	m, _ = m.Update(mouse(7, 0, tui.MouseActionMotion))
	if a, _ := m.Sizes(); a != 2 {
		t.Fatalf("rows top limit %d", a)
	}
}

func TestLinearize(t *testing.T) {
	m := newModel(21)
	m.Bounds = hittest.Rect{W: 21, H: 3}
	want := "Split pane, 2 columns, divider at 10 of 21\nPane 1:\nPane 2:"
	if got := m.Linearize(); got != want {
		t.Fatalf("Linearize =\n%s", got)
	}
	m.Direction = Rows
	m.Bounds = hittest.Rect{W: 5, H: 9}
	if got := m.Linearize(); !strings.HasPrefix(got, "Split pane, 2 rows, divider at 5 of 9") {
		t.Fatalf("Linearize = %q", got)
	}
}

type lin struct{ layout.Node }

func (lin) Linearize() string { return "inner text" }

func TestLinearizeIncludesPaneText(t *testing.T) {
	m := New(lin{layout.Block("a")}, layout.Block("b"))
	m.Total = 9
	if got := m.Linearize(); !strings.Contains(got, "Pane 1:\ninner text\nPane 2:") {
		t.Fatalf("Linearize = %q", got)
	}
}

func TestSetTheme(t *testing.T) {
	m := newModel(10)
	th := m.Theme
	th.BorderColor = ansi.Red
	a := m.View()
	m = m.SetTheme(th)
	if m.Theme.BorderColor != ansi.Red || m.View() == a {
		t.Fatal("theme not applied")
	}
}
