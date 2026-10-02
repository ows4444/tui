package clipboard

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

func mouseModel(out *strings.Builder) Model {
	m := New("secret", "Copy")
	m.Write = out.WriteString
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 6, H: 1}
	return m
}

func TestClickCopies(t *testing.T) {
	var out strings.Builder
	m, cmd := mouseModel(&out).Update(ev(7, 2, tui.MouseButtonLeft, tui.MouseActionPress))
	if !m.Copied() || cmd == nil || out.Len() == 0 {
		t.Fatalf("copied=%v cmd=%v written=%q", m.Copied(), cmd != nil, out.String())
	}
}

func TestClickIgnored(t *testing.T) {
	for name, e := range map[string]tui.MouseEvent{
		"miss":    ev(20, 2, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":  ev(7, 2, tui.MouseButtonRight, tui.MouseActionPress),
		"release": ev(7, 2, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		var out strings.Builder
		m, cmd := mouseModel(&out).Update(e)
		if m.Copied() || cmd != nil || out.Len() != 0 {
			t.Errorf("%s: acted", name)
		}
	}
}

func TestMouseOff(t *testing.T) {
	var out strings.Builder
	m := mouseModel(&out)
	m.Mouse = false
	if m, _ = m.Update(ev(7, 2, tui.MouseButtonLeft, tui.MouseActionPress)); m.Copied() || out.Len() != 0 {
		t.Fatal("Mouse off still copied")
	}
}
