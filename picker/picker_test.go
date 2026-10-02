package picker

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func TestNewStrings(t *testing.T) {
	m := NewStrings("a", "b", "c")
	if len(m.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(m.Items))
	}
	for i, s := range []string{"a", "b", "c"} {
		if m.Items[i].Label != s || m.Items[i].Value != s {
			t.Errorf("Items[%d] = %+v, want Label=Value=%q", i, m.Items[i], s)
		}
	}
}

func TestCursorMovement(t *testing.T) {
	m := NewStrings("a", "b", "c")

	next, _ := m.Update(key(tui.KeyDown))
	m = next
	if m.Cursor() != 1 {
		t.Fatalf("after Down: Cursor() = %d, want 1", m.Cursor())
	}

	next, _ = m.Update(key(tui.KeyDown))
	m = next
	next, _ = m.Update(key(tui.KeyDown)) // one past the end
	m = next
	if m.Cursor() != 2 {
		t.Fatalf("Down past the end: Cursor() = %d, want 2 (clamped)", m.Cursor())
	}

	next, _ = m.Update(key(tui.KeyUp))
	m = next
	if m.Cursor() != 1 {
		t.Fatalf("after Up: Cursor() = %d, want 1", m.Cursor())
	}

	m.SetCursor(0)
	next, _ = m.Update(key(tui.KeyUp)) // one past the start
	m = next
	if m.Cursor() != 0 {
		t.Fatalf("Up past the start: Cursor() = %d, want 0 (clamped)", m.Cursor())
	}

	next, _ = m.Update(key(tui.KeyEnd))
	m = next
	if m.Cursor() != 2 {
		t.Fatalf("after End: Cursor() = %d, want 2", m.Cursor())
	}
	next, _ = m.Update(key(tui.KeyHome))
	m = next
	if m.Cursor() != 0 {
		t.Fatalf("after Home: Cursor() = %d, want 0", m.Cursor())
	}
}

func TestSetCursorClamps(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.SetCursor(100)
	if m.Cursor() != 2 {
		t.Errorf("SetCursor(100) = %d, want 2 (clamped to last item)", m.Cursor())
	}
	m.SetCursor(-5)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor(-5) = %d, want 0 (clamped to first item)", m.Cursor())
	}
}

func TestEmptyModelDoesNotPanic(t *testing.T) {
	m := New()
	m.SetCursor(3)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor on empty Model = %d, want 0", m.Cursor())
	}
	if got := m.Highlighted(); got != (Item{}) {
		t.Errorf("Highlighted() on empty Model = %+v, want zero Item", got)
	}
	next, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil {
		t.Error("Enter on an empty Model should not emit a SelectedMsg")
	}
	if next.Cursor() != 0 {
		t.Errorf("Cursor() after Enter on empty Model = %d, want 0", next.Cursor())
	}
	if got := m.View(); got != "" {
		t.Errorf("View() on empty Model = %q, want empty", got)
	}
}

func TestEnterEmitsSelectedMsg(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.SetCursor(1)
	_, cmd := m.Update(key(tui.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	msg, ok := cmd().(SelectedMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want SelectedMsg", cmd())
	}
	if msg.Index != 1 || msg.Item.Label != "b" {
		t.Errorf("SelectedMsg = %+v, want {Index:1 Item:{b b}}", msg)
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := NewStrings("a", "b")
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if next.Cursor() != 0 || cmd != nil {
		t.Error("a non-Key Msg should be a no-op")
	}
}

func TestView(t *testing.T) {
	m := NewStrings("alpha", "beta")
	m.Theme = theme.DarkTheme()
	got := m.View()

	cursorStyle := theme.DarkTheme().ResolvedStates().Selected.Bold()
	want := "> " + cursorStyle.Render("alpha") + "\n  beta"
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewFollowsCursor(t *testing.T) {
	m := NewStrings("alpha", "beta")
	m.SetCursor(1)
	got := m.View()

	cursorStyle := m.Theme.ResolvedStates().Selected.Bold()
	want := "  alpha\n" + "> " + cursorStyle.Render("beta")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}
