package dialog

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func mouseDialog() Model {
	m := New("Title", "Message")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 3, Y: 2, W: 60, H: 20}
	return m
}

func TestClickOutsideDismisses(t *testing.T) {
	m := mouseDialog()
	got, cmd := m.Update(click(3, 2))
	if got.Open() || cmd == nil {
		t.Fatalf("open=%v cmd=%v", got.Open(), cmd != nil)
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("msg = %#v", cmd())
	}
}

func TestClickInsideDoesNotDismissAndIsSwallowed(t *testing.T) {
	m := mouseDialog()
	r := m.Rect()
	if r.W == 0 || !(hittest.Rect{X: 3, Y: 2, W: 60, H: 20}).Contains(r.X, r.Y) {
		t.Fatalf("Rect = %+v", r)
	}
	for _, ev := range []tui.MouseEvent{
		click(r.X+1, r.Y+1),
		{X: 0, Y: 0, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress},
		{X: 0, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease},
		{X: 0, Y: 0, Button: tui.MouseButtonRight, Action: tui.MouseActionPress},
	} {
		got, cmd := m.Update(ev)
		if !got.Open() || cmd != nil {
			t.Errorf("%+v dismissed the dialog", ev)
		}
	}
}

func TestRectMatchesRender(t *testing.T) {
	m := mouseDialog()
	m.Bounds = hittest.Rect{W: 40, H: 11}
	r := m.Rect()
	if r.X != (40-r.W)/2 || r.Y != (11-r.H)/2 {
		t.Errorf("Rect %+v is not centred in 40x11", r)
	}
	m.Hide()
	if m.Rect() != (hittest.Rect{}) {
		t.Error("closed dialog has a Rect")
	}
}

func TestMouseOffIgnoresClicksAndClosedIgnoresAll(t *testing.T) {
	m := mouseDialog()
	m.Mouse = false
	if got, _ := m.Update(click(3, 2)); !got.Open() {
		t.Error("mouse handled while Mouse is off")
	}
	m = mouseDialog()
	m.Hide()
	if _, cmd := m.Update(click(3, 2)); cmd != nil {
		t.Error("closed dialog reacted")
	}
}
