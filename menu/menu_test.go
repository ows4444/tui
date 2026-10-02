package menu

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/picker"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// testItems builds a 2-level menu: "Leaf A" (leaf), "Branch" (has children
// "Sub 1", "Sub 2"), "Leaf B" (leaf).
func testItems() []Item {
	return []Item{
		{Label: "Leaf A", Value: "a"},
		{Label: "Branch", Value: "branch", Children: []Item{
			{Label: "Sub 1", Value: "s1"},
			{Label: "Sub 2", Value: "s2"},
		}},
		{Label: "Leaf B", Value: "b"},
	}
}

// #403: with no drill-down happening, Menu behaves identically to
// picker.Model for Up/Down/Home/End navigation and rendering.
func TestNavigationMatchesPicker(t *testing.T) {
	items := testItems()
	m := New(items)

	pItems := make([]picker.Item, len(items))
	for i, it := range items {
		pItems[i] = picker.Item{Label: it.Label, Value: it.Value}
	}
	p := picker.New(pItems...)

	steps := []tui.KeyType{tui.KeyDown, tui.KeyDown, tui.KeyUp, tui.KeyEnd, tui.KeyHome, tui.KeyDown}
	for _, k := range steps {
		var mCmd tui.Cmd
		m, mCmd = m.Update(key(k))
		p, _ = p.Update(key(k))

		if mCmd != nil {
			t.Fatalf("unexpected Cmd from navigation key %v", k)
		}
		if m.View() != p.View() {
			t.Fatalf("after key %v: menu View() = %q, want %q (picker's)", k, m.View(), p.View())
		}
	}
}

// #404: Enter on an item with Children drills into a submenu instead of
// emitting a selection for the parent item.
func TestEnterOnBranchDrillsIn(t *testing.T) {
	m := New(testItems())

	// Move cursor to "Branch" (index 1).
	m, _ = m.Update(key(tui.KeyDown))
	if m.stack[m.current()].Highlighted().Label != "Branch" {
		t.Fatalf("expected cursor on Branch before Enter")
	}

	next, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SelectedMsg); ok {
			t.Fatalf("Enter on branch item emitted SelectedMsg, want no selection")
		}
	}
	if len(next.stack) != 2 {
		t.Fatalf("len(stack) = %d, want 2 after drilling into Branch", len(next.stack))
	}
	want := picker.New(picker.Item{Label: "Sub 1", Value: "s1"}, picker.Item{Label: "Sub 2", Value: "s2"}).View()
	if got := next.View(); got != want {
		t.Fatalf("View() after drill-in = %q, want %q (submenu rendering)", got, want)
	}
}

// #405: Enter on a leaf item (no Children) emits SelectedMsg identifying it.
func TestEnterOnLeafEmitsSelectedMsg(t *testing.T) {
	m := New(testItems())

	_, cmd := m.Update(key(tui.KeyEnter)) // cursor starts on "Leaf A"
	if cmd == nil {
		t.Fatalf("Enter on leaf item returned nil Cmd, want a Cmd delivering SelectedMsg")
	}
	msg := cmd()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("Cmd delivered %T, want SelectedMsg", msg)
	}
	if sel.Item.Label != "Leaf A" || sel.Item.Value != "a" {
		t.Fatalf("SelectedMsg.Item = %+v, want Leaf A", sel.Item)
	}
}

// #406: Esc inside a submenu pops back to the parent level, restoring its
// previous cursor position.
func TestEscPopsToParentPreservingCursor(t *testing.T) {
	m := New(testItems())

	// Move parent cursor to "Branch" (index 1), then drill in.
	m, _ = m.Update(key(tui.KeyDown))
	m, _ = m.Update(key(tui.KeyEnter))
	if len(m.stack) != 2 {
		t.Fatalf("setup: len(stack) = %d, want 2", len(m.stack))
	}

	// Move within the submenu so its own cursor differs from 0.
	m, _ = m.Update(key(tui.KeyDown))
	if m.stack[m.current()].Cursor() != 1 {
		t.Fatalf("setup: submenu cursor = %d, want 1", m.stack[m.current()].Cursor())
	}

	m, cmd := m.Update(key(tui.KeyEsc))
	if cmd != nil {
		t.Fatalf("Esc returned a non-nil Cmd, want nil")
	}
	if len(m.stack) != 1 {
		t.Fatalf("len(stack) = %d after Esc, want 1 (popped back to root)", len(m.stack))
	}
	if m.stack[m.current()].Cursor() != 1 {
		t.Fatalf("root cursor = %d after Esc, want 1 (preserved, still on Branch)", m.stack[m.current()].Cursor())
	}
	if m.stack[m.current()].Highlighted().Label != "Branch" {
		t.Fatalf("highlighted = %q after Esc, want Branch", m.stack[m.current()].Highlighted().Label)
	}
}

// #407: Esc at the root level (no parent to pop to) is a no-op, not a panic.
func TestEscAtRootIsNoOp(t *testing.T) {
	m := New(testItems())
	m, _ = m.Update(key(tui.KeyDown))

	next, cmd := m.Update(key(tui.KeyEsc))
	if cmd != nil {
		t.Fatalf("Esc at root returned a non-nil Cmd, want nil")
	}
	if len(next.stack) != 1 {
		t.Fatalf("len(stack) = %d after Esc at root, want 1 (unchanged)", len(next.stack))
	}
	if next.View() != m.View() {
		t.Fatalf("View() changed after no-op Esc at root: got %q, want %q", next.View(), m.View())
	}
}
