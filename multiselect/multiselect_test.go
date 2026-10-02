package multiselect

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func TestToggleAndIsSelected(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.Toggle(1)
	if !m.IsSelected(1) {
		t.Error("item 1 should be selected after Toggle(1)")
	}
	if m.IsSelected(0) || m.IsSelected(2) {
		t.Error("only item 1 should be selected")
	}
	m.Toggle(1)
	if m.IsSelected(1) {
		t.Error("Toggle(1) again should un-select it")
	}
}

// TestUpdateDoesNotMutateSharedSelectionMap is a regression test for the
// exact footgun Toggle's doc comment describes: since Model is used with
// value semantics (each Update call is meant to return an independent new
// Model), a naive `m.selected[i] = x` would mutate the map's shared
// underlying table in place, silently corrupting any other Model value
// still holding a copy of the same map header — including, concretely,
// the value the caller passed into Update in the first place.
func TestUpdateDoesNotMutateSharedSelectionMap(t *testing.T) {
	before := NewStrings("a", "b", "c")
	before.Toggle(0) // before now has {0: true}

	after, _ := before.Update(key(tui.KeySpace)) // toggles cursor (0) again -> {0: false}

	if !before.IsSelected(0) {
		t.Fatal("Update mutated the map shared with `before` — item 0 should still read selected on the original value")
	}
	if after.IsSelected(0) {
		t.Fatal("`after` should have item 0 un-selected (Space toggled it off)")
	}
}

func TestSelectedItemsAndIndexesAreSortedByPosition(t *testing.T) {
	m := NewStrings("a", "b", "c", "d")
	// Toggle out of order to make sure output order isn't just insertion
	// order (which, via a map, would be random anyway).
	m.Toggle(2)
	m.Toggle(0)
	m.Toggle(3)

	gotIdx := m.SelectedIndexes()
	wantIdx := []int{0, 2, 3}
	if len(gotIdx) != len(wantIdx) {
		t.Fatalf("SelectedIndexes() = %v, want %v", gotIdx, wantIdx)
	}
	for i := range wantIdx {
		if gotIdx[i] != wantIdx[i] {
			t.Fatalf("SelectedIndexes() = %v, want %v", gotIdx, wantIdx)
		}
	}

	gotItems := m.SelectedItems()
	wantLabels := []string{"a", "c", "d"}
	if len(gotItems) != len(wantLabels) {
		t.Fatalf("SelectedItems() = %v, want labels %v", gotItems, wantLabels)
	}
	for i, want := range wantLabels {
		if gotItems[i].Label != want {
			t.Errorf("SelectedItems()[%d].Label = %q, want %q", i, gotItems[i].Label, want)
		}
	}
}

func TestSpaceTogglesItemUnderCursor(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.SetCursor(1)
	next, _ := m.Update(key(tui.KeySpace))
	m = next
	if !m.IsSelected(1) {
		t.Fatal("Space should select the item under the cursor")
	}
	if m.IsSelected(0) || m.IsSelected(2) {
		t.Fatal("Space should only affect the item under the cursor")
	}
}

func TestSpaceOnEmptyModelDoesNotPanic(t *testing.T) {
	m := New()
	next, cmd := m.Update(key(tui.KeySpace))
	if cmd != nil {
		t.Error("Space on an empty Model should not return a Cmd")
	}
	if len(next.SelectedIndexes()) != 0 {
		t.Error("Space on an empty Model should not select anything")
	}
}

func TestEnterEmitsConfirmedMsgWithCurrentSelection(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.Toggle(0)
	m.Toggle(2)

	_, cmd := m.Update(key(tui.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	msg, ok := cmd().(ConfirmedMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want ConfirmedMsg", cmd())
	}
	if len(msg.Items) != 2 || msg.Items[0].Label != "a" || msg.Items[1].Label != "c" {
		t.Errorf("ConfirmedMsg.Items = %v, want [a c]", msg.Items)
	}
}

func TestEnterWithNothingSelectedEmitsEmptyConfirmedMsg(t *testing.T) {
	m := NewStrings("a", "b")
	_, cmd := m.Update(key(tui.KeyEnter))
	msg := cmd().(ConfirmedMsg)
	if len(msg.Items) != 0 {
		t.Errorf("ConfirmedMsg.Items = %v, want empty", msg.Items)
	}
}

func TestCursorMovementClamps(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.SetCursor(100)
	if m.Cursor() != 2 {
		t.Errorf("SetCursor(100) = %d, want 2", m.Cursor())
	}
	m.SetCursor(-5)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor(-5) = %d, want 0", m.Cursor())
	}
}

func TestSetCursorOnEmptyModel(t *testing.T) {
	m := New()
	m.SetCursor(5)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor(5) on an empty Model = %d, want 0", m.Cursor())
	}
}

func TestUpdateCursorNavigation(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		start int
		key   tui.KeyType
		want  int
	}{
		{"Up moves cursor back", []string{"a", "b", "c"}, 1, tui.KeyUp, 0},
		{"Up at top stays put", []string{"a", "b", "c"}, 0, tui.KeyUp, 0},
		{"Down moves cursor forward", []string{"a", "b", "c"}, 0, tui.KeyDown, 1},
		{"Down at bottom stays put", []string{"a", "b", "c"}, 2, tui.KeyDown, 2},
		{"Home jumps to first", []string{"a", "b", "c"}, 2, tui.KeyHome, 0},
		{"End jumps to last", []string{"a", "b", "c"}, 0, tui.KeyEnd, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewStrings(tt.items...)
			m.SetCursor(tt.start)
			next, cmd := m.Update(key(tt.key))
			if cmd != nil {
				t.Errorf("Update(%v) returned a non-nil Cmd", tt.key)
			}
			if next.Cursor() != tt.want {
				t.Errorf("Update(%v) cursor = %d, want %d", tt.key, next.Cursor(), tt.want)
			}
		})
	}
}

func TestUpdateHomeEndOnEmptyModelDoesNotPanic(t *testing.T) {
	m := New()
	if next, _ := m.Update(key(tui.KeyHome)); next.Cursor() != 0 {
		t.Errorf("Home on an empty Model cursor = %d, want 0", next.Cursor())
	}
	if next, _ := m.Update(key(tui.KeyEnd)); next.Cursor() != 0 {
		t.Errorf("End on an empty Model cursor = %d, want 0", next.Cursor())
	}
}

func TestUpdateIgnoresUnrecognizedKeyAndNonKeyMsg(t *testing.T) {
	m := NewStrings("a", "b")
	m.SetCursor(1)

	next, cmd := m.Update(key(tui.KeyTab))
	if cmd != nil {
		t.Error("an unrecognized key type should return a nil Cmd")
	}
	if next.Cursor() != 1 || next.IsSelected(0) || next.IsSelected(1) {
		t.Error("an unrecognized key type should leave the Model unchanged")
	}

	next, cmd = m.Update(struct{}{})
	if cmd != nil {
		t.Error("a non-Key Msg should return a nil Cmd")
	}
	if next.Cursor() != 1 {
		t.Error("a non-Key Msg should leave the Model unchanged")
	}
}

func TestView(t *testing.T) {
	m := NewStrings("alpha", "beta")
	m.Theme = theme.DarkTheme()
	m.Toggle(1)

	cursorStyle := theme.DarkTheme().ResolvedStates().Selected.Bold()
	checkStyle := ansi.NewStyle().Foreground(theme.DarkTheme().Success)

	got := m.View()
	want := cursorStyle.Render("> ") + "[ ] alpha\n" + "  " + checkStyle.Render("[x]") + " beta"
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}
