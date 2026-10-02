package accordion

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func sections() []Section {
	return []Section{
		{Title: "A", Content: "a-content"},
		{Title: "B", Content: "b-content"},
	}
}

func TestToggleAndIsExpanded(t *testing.T) {
	m := New(sections()...)
	m.Toggle(1)
	if !m.IsExpanded(1) {
		t.Error("section 1 should be expanded after Toggle(1)")
	}
	if m.IsExpanded(0) {
		t.Error("only section 1 should be expanded")
	}
	m.Toggle(1)
	if m.IsExpanded(1) {
		t.Error("Toggle(1) again should collapse it")
	}
}

// TestUpdateDoesNotMutateSharedExpandedMap mirrors
// multiselect's TestUpdateDoesNotMutateSharedSelectionMap: Toggle must
// rebuild the map rather than mutate it in place, or a value shared with
// an earlier Model copy would be silently corrupted.
func TestUpdateDoesNotMutateSharedExpandedMap(t *testing.T) {
	before := New(sections()...)
	before.Toggle(0) // before now has {0: true}

	after, _ := before.Update(key(tui.KeyEnter)) // toggles cursor (0) again -> {0: false}

	if !before.IsExpanded(0) {
		t.Fatal("Update mutated the map shared with `before` — section 0 should still read expanded on the original value")
	}
	if after.IsExpanded(0) {
		t.Fatal("`after` should have section 0 collapsed")
	}
}

func TestCursorMovement(t *testing.T) {
	m := New(sections()...)
	next, _ := m.Update(key(tui.KeyDown))
	m = next
	if m.Cursor() != 1 {
		t.Fatalf("after Down: Cursor() = %d, want 1", m.Cursor())
	}
	next, _ = m.Update(key(tui.KeyDown)) // one past the end
	m = next
	if m.Cursor() != 1 {
		t.Fatalf("Down past the end: Cursor() = %d, want 1 (clamped)", m.Cursor())
	}
	next, _ = m.Update(key(tui.KeyUp))
	m = next
	if m.Cursor() != 0 {
		t.Fatalf("after Up: Cursor() = %d, want 0", m.Cursor())
	}
}

func TestEnterAndSpaceBothToggle(t *testing.T) {
	m := New(sections()...)
	next, _ := m.Update(key(tui.KeyEnter))
	m = next
	if !m.IsExpanded(0) {
		t.Error("Enter should toggle the section under the cursor")
	}
	next, _ = m.Update(key(tui.KeySpace))
	m = next
	if m.IsExpanded(0) {
		t.Error("Space should toggle it back")
	}
}

func TestSetCursorClamps(t *testing.T) {
	m := New(sections()...)
	m.SetCursor(100)
	if m.Cursor() != 1 {
		t.Errorf("SetCursor(100) = %d, want 1", m.Cursor())
	}
	m.SetCursor(-5)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor(-5) = %d, want 0", m.Cursor())
	}
}

func TestEmptyModelDoesNotPanic(t *testing.T) {
	m := New()
	m.SetCursor(3)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor on empty Model = %d, want 0", m.Cursor())
	}
	next, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil || next.Cursor() != 0 {
		t.Error("Update on empty Model should be a safe no-op")
	}
	if got := m.View(); got != "" {
		t.Errorf("View() on empty Model = %q, want empty", got)
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := New(sections()...)
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.Cursor() != m.Cursor() {
		t.Error("a non-Key Msg should be a no-op")
	}
}

func TestView(t *testing.T) {
	dt := theme.DarkTheme()
	m := New(sections()...)
	m.Theme = dt
	m.Toggle(0) // expand section 0

	cursorStyle := dt.ResolvedStates().Selected.Bold()
	dim := ansi.NewStyle().Faint()

	want := "> ▾ " + cursorStyle.Render("A") + "\n    " + dim.Render("a-content") + "\n" + "  ▸ B"

	if got := m.View(); got != want {
		t.Errorf("View() =\n%q\nwant\n%q", got, want)
	}
}
