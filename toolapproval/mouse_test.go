package toolapproval

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

// Options row (row 2 with a description): " Approve " 0..8, " Deny " 11..16,
// " Always Allow " 19..32.
func mouseModel() Model {
	m := New("rm", "delete files", RiskHigh)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 10, Y: 4, W: 50, H: 3}
	return m
}

func TestClickChoices(t *testing.T) {
	for _, tc := range []struct {
		x    int
		want Choice
	}{{0, ChoiceApprove}, {8, ChoiceApprove}, {11, ChoiceDeny}, {16, ChoiceDeny}, {19, ChoiceAlwaysAllow}, {32, ChoiceAlwaysAllow}} {
		m, cmd := mouseModel().Update(ev(10+tc.x, 4+2, tui.MouseButtonLeft, tui.MouseActionPress))
		if cmd == nil {
			t.Fatalf("x=%d: no command", tc.x)
		}
		if r, ok := cmd().(ResolvedMsg); !ok || r.Choice != tc.want || m.Highlighted() != tc.want {
			t.Errorf("x=%d: got %+v, want %v", tc.x, r, tc.want)
		}
	}
}

func TestClickResolvesStaleTimeout(t *testing.T) {
	m := mouseModel()
	m.Start()
	m, _ = m.Update(ev(10+11, 4+2, tui.MouseButtonLeft, tui.MouseActionPress))
	if _, cmd := m.Update(timeoutMsg{id: 1}); cmd != nil {
		t.Error("timeout after a click resolved again")
	}
}

func TestClickIgnored(t *testing.T) {
	for name, e := range map[string]tui.MouseEvent{
		"gap":     ev(10+9, 6, tui.MouseButtonLeft, tui.MouseActionPress),
		"header":  ev(10+1, 4, tui.MouseButtonLeft, tui.MouseActionPress),
		"outside": ev(1, 6, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":  ev(10+1, 6, tui.MouseButtonRight, tui.MouseActionPress),
		"release": ev(10+1, 6, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		if _, cmd := mouseModel().Update(e); cmd != nil {
			t.Errorf("%s: produced a command", name)
		}
	}
}

func TestMouseOff(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	if _, cmd := m.Update(ev(10+1, 6, tui.MouseButtonLeft, tui.MouseActionPress)); cmd != nil {
		t.Fatal("Mouse off still resolved")
	}
}
