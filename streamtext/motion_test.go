package streamtext

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/motion"
)

func TestReducedMotionRevealsAtOnceAndKeepsTheCursorSolid(t *testing.T) {
	m := NewTypewriter()
	m.Motion = motion.Reduced
	if cmd := m.SetText("Hello. World"); cmd != nil {
		t.Fatal("SetText scheduled a tick")
	}
	if cmd := m.Append("!"); cmd != nil {
		t.Fatal("Append scheduled a tick")
	}
	if !m.Done() || !strings.Contains(m.View(), "Hello. World!") {
		t.Errorf("text should be fully shown, got %q", m.View())
	}
	if m.blinkOff {
		t.Error("the cursor should stay solid")
	}
}
