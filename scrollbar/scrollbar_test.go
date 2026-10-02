package scrollbar

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestThumbSizeAndPosition(t *testing.T) {
	m := New(100, 25)
	if s, n := m.Thumb(20); s != 0 || n != 5 {
		t.Fatalf("top: %d,%d", s, n)
	}
	m.SetOffset(m.MaxOffset())
	if s, n := m.Thumb(20); s != 15 || n != 5 {
		t.Fatalf("end: %d,%d", s, n)
	}
	m.SetOffset(37) // half of 75 rounds to the middle of the 15 free cells
	if s, _ := m.Thumb(20); s < 7 || s > 8 {
		t.Fatalf("middle start %d", s)
	}
	// Tiny visible fraction keeps a one-cell thumb; everything visible fills.
	if _, n := New(10000, 1).Thumb(10); n != 1 {
		t.Fatalf("min thumb %d", n)
	}
	if s, n := New(5, 10).Thumb(8); s != 0 || n != 8 {
		t.Fatalf("all visible: %d,%d", s, n)
	}
	if s, n := m.Thumb(0); s != 0 || n != 0 {
		t.Fatalf("empty track: %d,%d", s, n)
	}
}

func TestSetOffsetClamps(t *testing.T) {
	m := New(50, 10)
	m.SetOffset(99)
	if m.Offset != 40 {
		t.Fatalf("offset %d", m.Offset)
	}
	m.SetOffset(-5)
	if m.Offset != 0 {
		t.Fatalf("offset %d", m.Offset)
	}
	m.SetOffset(40)
	m.SetContent(20, 10)
	if m.Offset != 10 {
		t.Fatalf("SetContent offset %d", m.Offset)
	}
}

func TestKeys(t *testing.T) {
	m := New(100, 10)
	m, cmd := m.Update(key(tui.KeyDown))
	if m.Offset != 1 {
		t.Fatalf("down: %d", m.Offset)
	}
	if msg, _ := cmd().(ScrolledMsg); msg.Offset != 1 {
		t.Fatalf("msg %+v", msg)
	}
	m, _ = m.Update(key(tui.KeyPgDown))
	if m.Offset != 11 {
		t.Fatalf("pgdown: %d", m.Offset)
	}
	m, _ = m.Update(key(tui.KeyEnd))
	if m.Offset != 90 {
		t.Fatalf("end: %d", m.Offset)
	}
	m, cmd = m.Update(key(tui.KeyDown))
	if cmd != nil {
		t.Fatal("no change must not send a message")
	}
	m, _ = m.Update(key(tui.KeyHome))
	if m.Offset != 0 {
		t.Fatalf("home: %d", m.Offset)
	}
	if _, cmd = m.Update("x"); cmd != nil {
		t.Fatal("other msg should be a no-op")
	}
}

func TestRebindAndFallback(t *testing.T) {
	m := New(100, 10)
	m.KeyMap.Down.Keys = []string{"n"}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Offset != 0 {
		t.Fatalf("arrow scrolled after rebind: %d", m.Offset)
	}
	m, _ = m.Update(runeKey('n'))
	if m.Offset != 1 {
		t.Fatalf("n: %d", m.Offset)
	}
	lit := Model{Total: 100, Visible: 10}
	lit, _ = lit.Update(key(tui.KeyDown))
	if lit.Offset != 1 {
		t.Fatalf("literal Model: %d", lit.Offset)
	}
	for _, b := range New(1, 1).Bindings() {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

func mouse(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

func TestMouseClickDragWheel(t *testing.T) {
	m := New(100, 20) // track 10 rows, thumb 2 cells
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 1, H: 10}
	press := func(m Model, y int) (Model, tui.Cmd) {
		return m.Update(mouse(5, y, tui.MouseButtonLeft, tui.MouseActionPress))
	}
	m2, _ := press(m, 2)
	if m2.Dragging() || m2.Offset != 0 {
		t.Fatal("Mouse off must ignore the mouse")
	}
	m.Mouse = true

	// Click the track below the thumb: one page down.
	m, cmd := press(m, 8)
	if m.Offset != 20 || cmd == nil {
		t.Fatalf("page click: %d", m.Offset)
	}
	// Click above the thumb: back up a page.
	m, _ = press(m, 2)
	if m.Offset != 0 {
		t.Fatalf("page up click: %d", m.Offset)
	}
	// Outside Bounds: ignored.
	if m3, _ := m.Update(mouse(9, 4, tui.MouseButtonLeft, tui.MouseActionPress)); m3.Dragging() {
		t.Fatal("outside press started a drag")
	}

	// Drag the thumb (rows 2-3) to the bottom, even past the bar.
	m, _ = press(m, 2)
	if !m.Dragging() {
		t.Fatal("press on thumb did not start drag")
	}
	m, cmd = m.Update(mouse(5, 6, tui.MouseButtonLeft, tui.MouseActionMotion))
	if m.Offset <= 0 || m.Offset >= 80 || cmd == nil {
		t.Fatalf("mid drag offset %d", m.Offset)
	}
	m, _ = m.Update(mouse(40, 99, tui.MouseButtonLeft, tui.MouseActionMotion))
	if m.Offset != 80 {
		t.Fatalf("drag past end: %d", m.Offset)
	}
	m, _ = m.Update(mouse(5, 99, tui.MouseButtonLeft, tui.MouseActionRelease))
	if m.Dragging() {
		t.Fatal("release did not end drag")
	}
	m, _ = m.Update(mouse(5, 5, tui.MouseButtonLeft, tui.MouseActionMotion))
	if m.Offset != 80 {
		t.Fatalf("motion after release moved: %d", m.Offset)
	}

	m, _ = m.Update(mouse(5, 5, tui.MouseButtonWheelUp, tui.MouseActionPress))
	if m.Offset != 77 {
		t.Fatalf("wheel up: %d", m.Offset)
	}
	m.Wheel = 1
	m, _ = m.Update(mouse(5, 5, tui.MouseButtonWheelDown, tui.MouseActionPress))
	if m.Offset != 78 {
		t.Fatalf("wheel down: %d", m.Offset)
	}
}

func TestHorizontalMouseDrag(t *testing.T) {
	m := New(40, 10)
	m.Orientation = Horizontal
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 5, W: 20, H: 1}
	m, _ = m.Update(mouse(1, 5, tui.MouseButtonLeft, tui.MouseActionPress))
	if !m.Dragging() {
		t.Fatal("no drag")
	}
	m, _ = m.Update(mouse(19, 5, tui.MouseButtonLeft, tui.MouseActionMotion))
	if m.Offset != 30 {
		t.Fatalf("offset %d", m.Offset)
	}
}

func TestViewAndLinearize(t *testing.T) {
	m := New(40, 10)
	m.Length = 8
	rows := strings.Split(ansi.StripANSI(m.View()), "\n")
	if len(rows) != 8 || rows[0] != "█" || rows[1] != "█" || rows[7] != "░" {
		t.Fatalf("rows %q", rows)
	}
	m.Orientation = Horizontal
	if got := ansi.StripANSI(m.View()); got != "██░░░░░░" {
		t.Fatalf("horizontal %q", got)
	}
	if got := m.Linearize(); got != "Scrollbar, horizontal, 0% scrolled, showing 10 of 40" {
		t.Fatalf("Linearize %q", got)
	}
	if got := New(5, 10).Linearize(); got != "Scrollbar, vertical, all content visible" {
		t.Fatalf("Linearize %q", got)
	}
}

func TestSetTheme(t *testing.T) {
	m := New(1, 1)
	th := m.Theme
	th.Primary = ansi.Red
	if got := m.SetTheme(th); got.Theme.Primary != ansi.Red {
		t.Fatal("theme not applied")
	}
	m.Length = 3
	a := m.View()
	m = m.SetTheme(th)
	if m.View() == a {
		t.Fatal("theme change did not alter styling")
	}
}
