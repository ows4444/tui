package popover

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func mouseModel() Model {
	m := New("content", 5, 2)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 1, Y: 1, W: 60, H: 20}
	return m
}

func TestClickOutsideDismisses(t *testing.T) {
	m := mouseModel()
	r := m.Rect()
	if r.W == 0 || r.H == 0 {
		t.Fatalf("Rect = %+v", r)
	}
	x, y := 1, 1 // Bounds origin; the box never starts there with these anchors
	if r.Contains(x, y) {
		x, y = 1+59, 1+19
	}
	got, cmd := m.Update(click(x, y))
	if got.Open() || cmd == nil {
		t.Fatalf("open=%v cmd=%v", got.Open(), cmd != nil)
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("msg = %#v", cmd())
	}
}

func TestClickInsideAndOtherEventsAreSwallowed(t *testing.T) {
	m := mouseModel()
	r := m.Rect()
	for _, ev := range []tui.MouseEvent{
		click(r.X+1, r.Y+1),
		{X: 1, Y: 1, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress},
		{X: 1, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease},
		{X: 1, Y: 1, Button: tui.MouseButtonRight, Action: tui.MouseActionPress},
	} {
		if got, cmd := m.Update(ev); !got.Open() || cmd != nil {
			t.Errorf("%+v dismissed it", ev)
		}
	}
}

func TestMouseOffAndClosed(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	if got, _ := m.Update(click(1, 1)); !got.Open() {
		t.Error("mouse handled while Mouse is off")
	}
	m = mouseModel()
	m.Hide()
	if got, cmd := m.Update(click(1, 1)); got.Open() || cmd != nil || m.Rect() != (hittest.Rect{}) {
		t.Error("closed model reacted")
	}
}
