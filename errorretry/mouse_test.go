package errorretry

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

// Hint "press Enter or r to retry (0/2)" is 30 wide, then ", Esc to dismiss".
func mouseModel() Model {
	m := New("boom", 2)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 5, Y: 3, W: 60, H: 2}
	return m
}

func TestClickRetry(t *testing.T) {
	m, cmd := mouseModel().Update(ev(5+4, 3+1, tui.MouseButtonLeft, tui.MouseActionPress))
	if cmd == nil {
		t.Fatal("no command")
	}
	if _, ok := cmd().(RetryMsg); !ok || m.RetryCount() != 1 {
		t.Fatalf("retry click: count=%d", m.RetryCount())
	}
}

func TestClickDismiss(t *testing.T) {
	_, cmd := mouseModel().Update(ev(5+35, 3+1, tui.MouseButtonLeft, tui.MouseActionPress))
	if cmd == nil {
		t.Fatal("no command")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Fatal("not DismissedMsg")
	}
}

func TestExhaustedHintDismissesOnly(t *testing.T) {
	m := mouseModel()
	m.MaxRetries = 0
	_, cmd := m.Update(ev(5+2, 3+1, tui.MouseButtonLeft, tui.MouseActionPress))
	if cmd == nil {
		t.Fatal("no command")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Fatal("exhausted click should dismiss")
	}
}

func TestClickIgnored(t *testing.T) {
	for name, e := range map[string]tui.MouseEvent{
		"message":   ev(5+4, 3, tui.MouseButtonLeft, tui.MouseActionPress),
		"outside":   ev(0, 4, tui.MouseButtonLeft, tui.MouseActionPress),
		"past line": ev(5+55, 4, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":    ev(5+4, 4, tui.MouseButtonRight, tui.MouseActionPress),
		"release":   ev(5+4, 4, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		if _, cmd := mouseModel().Update(e); cmd != nil {
			t.Errorf("%s: produced a command", name)
		}
	}
}

func TestMouseOff(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	if _, cmd := m.Update(ev(5+4, 4, tui.MouseButtonLeft, tui.MouseActionPress)); cmd != nil {
		t.Fatal("Mouse off still acted")
	}
}
