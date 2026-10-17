package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func key(m model, k tui.Key) (model, tui.Cmd) {
	next, cmd := m.Update(k)
	return next.(model), cmd
}

func mouse(m model, ev tui.MouseEvent) model {
	next, _ := m.Update(ev)
	return next.(model)
}

// centre is a cell inside the named target, read from the drawn layout.
func centre(t testing.TB, m model, id string) (x, y int) {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen", id)
	}
	return r.X + r.W/2, r.Y + r.H/2
}

// click presses and releases the left button on the named target.
func click(m model, id string) model {
	r, _ := layout.RectOf(m.screen(), m.size(), id)
	x, y := r.X+r.W/2, r.Y+r.H/2
	m = mouse(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	return mouse(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
}

func TestClickButtons(t *testing.T) {
	m := initialModel()
	m = click(m, "inc")
	m = click(m, "inc")
	m = click(m, "dec")
	if m.count != 1 {
		t.Fatalf("count = %d after + + -, want 1", m.count)
	}
	m = click(m, "reset")
	if m.count != 0 {
		t.Fatalf("count = %d after Reset, want 0", m.count)
	}
}

func TestStepChipsChangeTheStep(t *testing.T) {
	m := click(initialModel(), "step-2")
	m = click(m, "inc")
	if m.count != 10 {
		t.Fatalf("count = %d with step 10, want 10", m.count)
	}
}

func TestReleaseElsewhereCancelsTheClick(t *testing.T) {
	m := initialModel()
	x, y := centre(t, m, "inc")
	m = mouse(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	m = mouse(m, tui.MouseEvent{X: 0, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	if m.count != 0 || m.pressed != "" {
		t.Fatalf("count = %d, pressed = %q; a drag off the button must cancel", m.count, m.pressed)
	}
}

func TestWheelChangesTheCount(t *testing.T) {
	m := mouse(initialModel(), tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	m = mouse(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	m = mouse(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.count != 1 {
		t.Fatalf("count = %d, want 1", m.count)
	}
}

func TestHoverFollowsThePointer(t *testing.T) {
	m := initialModel()
	x, y := centre(t, m, "reset")
	m = mouse(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
	if m.hover != "reset" {
		t.Fatalf("hover = %q, want reset", m.hover)
	}
	m = mouse(m, tui.MouseEvent{X: 0, Y: 0, Action: tui.MouseActionMotion})
	if m.hover != "" {
		t.Fatalf("hover = %q off the panel, want none", m.hover)
	}
}

func TestKeys(t *testing.T) {
	m := initialModel()
	for _, k := range []tui.Key{
		{Type: tui.KeyUp}, {Type: tui.KeyRunes, Text: "+"}, {Type: tui.KeyRight}, {Type: tui.KeyDown},
	} {
		m, _ = key(m, k)
	}
	if m.count != 2 {
		t.Fatalf("count = %d, want 2", m.count)
	}
	m, _ = key(m, tui.Key{Type: tui.KeyRunes, Text: "s"})
	m, _ = key(m, tui.Key{Type: tui.KeyUp})
	if m.count != 7 {
		t.Fatalf("count = %d after s and up, want 7", m.count)
	}
	m, _ = key(m, tui.Key{Type: tui.KeyRunes, Text: "r"})
	if m.count != 0 {
		t.Fatalf("count = %d after r, want 0", m.count)
	}
}

func TestCountIsClamped(t *testing.T) {
	m := initialModel()
	m.step = 2
	for i := 0; i < 30; i++ {
		m = m.add(steps[m.step])
	}
	if m.count != bound {
		t.Fatalf("count = %d, want the bound %d", m.count, bound)
	}
}

func TestHelpToggleAndQuit(t *testing.T) {
	m, _ := key(initialModel(), tui.Key{Type: tui.KeyRunes, Text: "?"})
	if !m.help || !strings.Contains(ansi.StripANSI(m.View()), "next step size") {
		t.Fatal("? did not open the help")
	}
	m, cmd := key(m, tui.Key{Type: tui.KeyEsc})
	if m.help || cmd != nil {
		t.Fatal("Esc must close the help and not quit")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyEsc}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := key(initialModel(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestViewShowsTheCount(t *testing.T) {
	m := initialModel().add(7)
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "+7") || !strings.Contains(out, "Reset") {
		t.Fatalf("View missing the count or Reset:\n%s", out)
	}
}
