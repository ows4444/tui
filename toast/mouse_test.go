package toast

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func shown() Model {
	m := New("saved")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 2, Y: 1, W: 40, H: 10}
	m.Show()
	return m
}

func TestClickOnToastDismisses(t *testing.T) {
	m := shown()
	r := m.Rect()
	if r.W == 0 || r.X < 2 || r.X+r.W > 42 || r.Y+r.H > 11 {
		t.Fatalf("Rect %+v not inside Bounds", r)
	}
	got, cmd := m.Update(click(r.X+1, r.Y+1))
	if got.Open() || cmd == nil {
		t.Fatalf("open=%v cmd=%v", got.Open(), cmd != nil)
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("msg = %#v", cmd())
	}
}

func TestClickElsewhereWrongButtonOrOffIgnored(t *testing.T) {
	m := shown()
	r := m.Rect()
	for name, ev := range map[string]tui.MouseEvent{
		"outside":     click(2, 1),
		"right click": {X: r.X + 1, Y: r.Y + 1, Button: tui.MouseButtonRight, Action: tui.MouseActionPress},
		"release":     {X: r.X + 1, Y: r.Y + 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease},
	} {
		if got, cmd := m.Update(ev); !got.Open() || cmd != nil {
			t.Errorf("%s dismissed the toast", name)
		}
	}
	m.Mouse = false
	if got, _ := m.Update(click(r.X+1, r.Y+1)); !got.Open() {
		t.Error("mouse handled while Mouse is off")
	}
}

func TestRectFollowsPositionAndClosedHasNone(t *testing.T) {
	m := shown()
	m.Position = TopLeft
	if r := m.Rect(); r.X != 3 || r.Y != 2 {
		t.Errorf("TopLeft rect = %+v, want origin (3,2)", r)
	}
	m.Hide()
	if m.Rect() != (hittest.Rect{}) {
		t.Error("closed toast has a Rect")
	}
}
