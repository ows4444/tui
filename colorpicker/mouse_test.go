package colorpicker

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
)

func press(x, y int, b tui.MouseButton) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress}
}

func mouseModel() Model {
	m := New(ansi.RGB{R: 255}, ansi.RGB{G: 255}, ansi.RGB{B: 255})
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 4, Y: 1, W: 40, H: 1}
	return m
}

func TestClickSwatchMovesCursorThenConfirms(t *testing.T) {
	m := mouseModel()
	m, cmd := m.Update(press(4+3, 1, tui.MouseButtonLeft)) // swatch 1
	if m.Cursor() != 1 || cmd != nil {
		t.Fatalf("cursor=%d cmd=%v", m.Cursor(), cmd != nil)
	}
	_, cmd = m.Update(press(4+4, 1, tui.MouseButtonLeft)) // swatch 1 again, 2nd cell
	if cmd == nil {
		t.Fatal("click on the highlighted swatch did not confirm")
	}
	if got, ok := cmd().(SelectedMsg); !ok || got.Color != (ansi.RGB{G: 255}) {
		t.Errorf("msg = %#v", cmd())
	}
}

func TestClickGapOutsideAndWrongButtonIgnored(t *testing.T) {
	m := mouseModel()
	for _, ev := range []tui.MouseEvent{
		press(4+2, 1, tui.MouseButtonLeft),  // gap between swatches 0 and 1
		press(0, 1, tui.MouseButtonLeft),    // outside Bounds
		press(4+3, 1, tui.MouseButtonRight), // wrong button
	} {
		got, cmd := m.Update(ev)
		if got.Cursor() != 0 || cmd != nil || got.HexFocused() {
			t.Errorf("%+v changed state", ev)
		}
	}
}

func TestClickHexInputFocusesItAndSwatchReturnsToPalette(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(press(4+3*3+2, 1, tui.MouseButtonLeft))
	if !m.HexFocused() {
		t.Fatal("click on the hex input did not focus it")
	}
	m, cmd := m.Update(press(4+3*2, 1, tui.MouseButtonLeft)) // swatch 2
	if m.HexFocused() || m.Cursor() != 2 || cmd != nil {
		t.Errorf("hex=%v cursor=%d cmd=%v: a swatch click from the hex pane must only move back", m.HexFocused(), m.Cursor(), cmd != nil)
	}
}

func TestWheelMovesCursor(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(press(5, 1, tui.MouseButtonWheelDown))
	m, _ = m.Update(press(5, 1, tui.MouseButtonWheelDown))
	m, _ = m.Update(press(5, 1, tui.MouseButtonWheelDown))
	if m.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", m.Cursor())
	}
	m, _ = m.Update(press(5, 1, tui.MouseButtonWheelUp))
	if m.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor())
	}
}

func TestMouseOffIgnoresEvents(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	if got, cmd := m.Update(press(4+3, 1, tui.MouseButtonLeft)); got.Cursor() != 0 || cmd != nil {
		t.Error("mouse handled while Mouse is off")
	}
}
