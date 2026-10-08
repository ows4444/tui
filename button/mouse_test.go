package button

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

// A button drawn at column 10, row 4: "[ Save ]" is 8 cells.
func mouseModel() Model {
	m := New("Save")
	m.ID = "save"
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 10, Y: 4, W: 8, H: 1}
	return m
}

func TestAClickIsAPressAndAReleaseInside(t *testing.T) {
	m := mouseModel()
	m, cmd := m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	if cmd != nil {
		t.Fatal("the button fired when the pointer went down, before it came up")
	}
	if !m.Down() {
		t.Fatal("Down() is false while the pointer is held on the button")
	}
	m, cmd = m.Update(ev(13, 4, tui.MouseButtonLeft, tui.MouseActionRelease))
	if p, ok := pressed(cmd); !ok || p.ID != "save" {
		t.Fatalf("release inside: PressedMsg = %+v, delivered = %v", p, ok)
	}
	if m.Down() {
		t.Fatal("Down() is still true after the release")
	}
}

// Dragging off the button before letting go is how a press is taken back.
func TestAReleaseOutsidePressesNothing(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	m, cmd := m.Update(ev(30, 4, tui.MouseButtonLeft, tui.MouseActionRelease))
	if cmd != nil {
		t.Fatal("a release outside the button pressed it")
	}
	if m.Down() {
		t.Fatal("Down() is still true after the release")
	}
}

func TestAPressOutsideThenAReleaseInsidePressesNothing(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(ev(30, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	if _, cmd := m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionRelease)); cmd != nil {
		t.Fatal("a release inside pressed a button the pointer did not go down on")
	}
}

func TestOnlyTheLeftButtonPresses(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(ev(12, 4, tui.MouseButtonRight, tui.MouseActionPress))
	if m.Down() {
		t.Fatal("the right button went down on it")
	}
}

func TestMotionSetsAndClearsHover(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(ev(12, 4, tui.MouseButtonNone, tui.MouseActionMotion))
	if !m.Hovered() {
		t.Fatal("Hovered() is false with the pointer over the button")
	}
	m, _ = m.Update(ev(12, 5, tui.MouseButtonNone, tui.MouseActionMotion))
	if m.Hovered() {
		t.Fatal("Hovered() is still true after the pointer left")
	}
}

func TestMouseOffIgnoresThePointer(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	m, _ = m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	next, cmd := m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionRelease))
	if cmd != nil || next.Down() || next.Hovered() {
		t.Fatal("the pointer acted on a button with Mouse off")
	}
}

func TestBlurLetsGoOfAPressInProgress(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	m.Blur()
	if _, cmd := m.Update(ev(12, 4, tui.MouseButtonLeft, tui.MouseActionRelease)); cmd != nil {
		t.Fatal("a release pressed the button after Blur let go of the press")
	}
}
