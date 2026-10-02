package textinput

import (
	"testing"

	"github.com/ows4444/tui/motion"
)

func TestReducedMotionKeepsTheCursorSolid(t *testing.T) {
	m := New()
	m.Motion = motion.Reduced
	if cmd := m.Focus(); cmd != nil {
		t.Fatal("Focus scheduled a blink")
	}
	if !m.cursorVisible {
		t.Fatal("cursor should be visible after Focus")
	}
	m.cursorVisible = false // a stray blink from before the preference changed
	m, cmd := m.Update(blinkMsg{id: m.blinkID})
	if cmd != nil || !m.cursorVisible {
		t.Errorf("blink under reduced motion: cmd %v, cursorVisible %v; want no tick and a solid cursor", cmd, m.cursorVisible)
	}
}

func TestNormalMotionStillBlinks(t *testing.T) {
	m := New()
	m.Focus()
	m, cmd := m.Update(blinkMsg{id: m.blinkID})
	if cmd == nil || m.cursorVisible {
		t.Errorf("blink under normal motion: cmd %v, cursorVisible %v; want a tick and a hidden cursor", cmd, m.cursorVisible)
	}
}
