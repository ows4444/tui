package confirm

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

// "Go?" is 3 wide: Yes label cells 5..9 (" Yes "), No label cells 12..15.
func mouseModel() Model {
	m := New("Go?")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 10, Y: 4, W: 40, H: 1}
	return m
}

func answer(t *testing.T, cmd tui.Cmd) (ConfirmedMsg, bool) {
	t.Helper()
	if cmd == nil {
		return ConfirmedMsg{}, false
	}
	c, ok := cmd().(ConfirmedMsg)
	return c, ok
}

func TestClickYesAndNo(t *testing.T) {
	m, cmd := mouseModel().Update(ev(10+5, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	if c, ok := answer(t, cmd); !ok || !c.Yes {
		t.Fatalf("yes click: %+v %v", c, ok)
	}
	m, cmd = m.Update(ev(10+12, 4, tui.MouseButtonLeft, tui.MouseActionPress))
	if c, ok := answer(t, cmd); !ok || c.Yes {
		t.Fatalf("no click: %+v %v", c, ok)
	}
	if m.Highlighted() {
		t.Error("highlight did not follow the No click")
	}
}

func TestClickIgnored(t *testing.T) {
	for name, e := range map[string]tui.MouseEvent{
		"gap":     ev(10+10, 4, tui.MouseButtonLeft, tui.MouseActionPress),
		"prompt":  ev(10+1, 4, tui.MouseButtonLeft, tui.MouseActionPress),
		"outside": ev(2, 4, tui.MouseButtonLeft, tui.MouseActionPress),
		"row":     ev(10+5, 5, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":  ev(10+5, 4, tui.MouseButtonRight, tui.MouseActionPress),
		"release": ev(10+5, 4, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		if _, cmd := mouseModel().Update(e); cmd != nil {
			t.Errorf("%s: produced a command", name)
		}
	}
}

func TestMouseOff(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	if _, cmd := m.Update(ev(10+5, 4, tui.MouseButtonLeft, tui.MouseActionPress)); cmd != nil {
		t.Fatal("Mouse off still answered")
	}
}
