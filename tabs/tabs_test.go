package tabs

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func TestNewStartsAtZero(t *testing.T) {
	m := New("A", "B", "C")
	if m.Active() != 0 {
		t.Errorf("Active() = %d, want 0", m.Active())
	}
}

func TestLeftRightMoveAndClamp(t *testing.T) {
	m := New("A", "B", "C")

	next, cmd := m.Update(key(tui.KeyRight))
	m = next
	if m.Active() != 1 || cmd == nil {
		t.Fatalf("after Right: Active()=%d cmd=%v, want 1, non-nil", m.Active(), cmd)
	}
	msg := cmd().(ChangedMsg)
	if msg.Index != 1 {
		t.Errorf("ChangedMsg.Index = %d, want 1", msg.Index)
	}

	next, _ = m.Update(key(tui.KeyRight))
	m = next
	next, cmd = m.Update(key(tui.KeyRight)) // one past the end
	m = next
	if m.Active() != 2 || cmd != nil {
		t.Fatalf("Right past the end: Active()=%d cmd=%v, want 2, nil (clamped, no Cmd)", m.Active(), cmd)
	}

	next, cmd = m.Update(key(tui.KeyLeft))
	m = next
	if m.Active() != 1 || cmd == nil {
		t.Fatalf("after Left: Active()=%d cmd=%v, want 1, non-nil", m.Active(), cmd)
	}

	m.SetActive(0)
	next, cmd = m.Update(key(tui.KeyLeft)) // one past the start
	m = next
	if m.Active() != 0 || cmd != nil {
		t.Fatalf("Left past the start: Active()=%d cmd=%v, want 0, nil", m.Active(), cmd)
	}
}

func TestTabWraps(t *testing.T) {
	m := New("A", "B", "C")
	m.SetActive(2)
	next, cmd := m.Update(key(tui.KeyTab))
	m = next
	if m.Active() != 0 || cmd == nil {
		t.Fatalf("Tab from the last index: Active()=%d cmd=%v, want 0, non-nil (wraps)", m.Active(), cmd)
	}
}

func TestSetActiveClamps(t *testing.T) {
	m := New("A", "B")
	m.SetActive(100)
	if m.Active() != 1 {
		t.Errorf("SetActive(100) = %d, want 1", m.Active())
	}
	m.SetActive(-5)
	if m.Active() != 0 {
		t.Errorf("SetActive(-5) = %d, want 0", m.Active())
	}
}

func TestEmptyModelDoesNotPanic(t *testing.T) {
	m := New()
	m.SetActive(3)
	if m.Active() != 0 {
		t.Errorf("SetActive on empty Model = %d, want 0", m.Active())
	}
	next, cmd := m.Update(key(tui.KeyRight))
	if cmd != nil || next.Active() != 0 {
		t.Error("Update on empty Model should be a safe no-op")
	}
	if got := m.View(); got != "" {
		t.Errorf("View() on empty Model = %q, want empty", got)
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := New("A", "B")
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.Active() != m.Active() {
		t.Error("a non-Key Msg should be a no-op")
	}
}

func TestView(t *testing.T) {
	dt := theme.DarkTheme()
	m := New("One", "Two")
	m.Theme = dt

	active := dt.ResolvedStates().Selected.Bold()
	inactive := ansi.NewStyle().Foreground(dt.Muted)
	want := active.Render("[One]") + " " + inactive.Render(" Two ")

	if got := m.View(); got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}
