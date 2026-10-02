package treeview

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func testTree() Model {
	return New(Node{
		Label: "root",
		Children: []Node{
			{Label: "child0", Children: []Node{
				{Label: "grandchild0"},
			}},
			{Label: "child1"},
		},
	})
}

func TestFlattenCollapsedShowsOnlyRoot(t *testing.T) {
	m := testTree()
	items := m.flatten()
	if len(items) != 1 || items[0].node.Label != "root" {
		t.Fatalf("flatten() with nothing expanded = %v, want just [root]", labels(items))
	}
}

func labels(items []visible) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.node.Label
	}
	return out
}

func TestExpandRevealsChildren(t *testing.T) {
	m := testTree()
	m.toggle("0") // expand root
	items := m.flatten()
	want := []string{"root", "child0", "child1"}
	got := labels(items)
	if len(got) != len(want) {
		t.Fatalf("flatten() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("flatten() = %v, want %v", got, want)
		}
	}
	// grandchild0 shouldn't show yet: child0 itself isn't expanded.
	for _, l := range got {
		if l == "grandchild0" {
			t.Fatal("grandchild0 should not be visible until child0 is expanded")
		}
	}
}

func TestExpandNestedRevealsGrandchild(t *testing.T) {
	m := testTree()
	m.toggle("0")   // expand root
	m.toggle("0.0") // expand child0
	got := labels(m.flatten())
	want := []string{"root", "child0", "grandchild0", "child1"}
	if len(got) != len(want) {
		t.Fatalf("flatten() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("flatten() = %v, want %v", got, want)
		}
	}
}

// TestUpdateDoesNotMutateSharedExpandedMap mirrors the same regression
// test in multiselect/accordion: toggle must rebuild the map rather than
// mutate it in place.
func TestUpdateDoesNotMutateSharedExpandedMap(t *testing.T) {
	before := testTree()
	before.toggle("0") // before now has {"0": true}

	after, _ := before.Update(key(tui.KeyEnter)) // toggles cursor (path "0") again -> collapsed

	if !before.IsExpanded("0") {
		t.Fatal("Update mutated the map shared with `before` — root should still read expanded on the original value")
	}
	if after.IsExpanded("0") {
		t.Fatal("`after` should have root collapsed")
	}
}

func TestEnterOnLeafEmitsSelectedMsg(t *testing.T) {
	m := testTree()
	next, _ := m.Update(key(tui.KeyEnter)) // expands root
	m = next
	m.SetCursor(2) // "child1", a leaf

	_, cmd := m.Update(key(tui.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter on a leaf should return a non-nil Cmd")
	}
	msg, ok := cmd().(SelectedMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want SelectedMsg", cmd())
	}
	if msg.Node.Label != "child1" {
		t.Errorf("SelectedMsg.Node.Label = %q, want %q", msg.Node.Label, "child1")
	}
}

func TestEnterOnBranchTogglesNotSelects(t *testing.T) {
	m := testTree()
	_, cmd := m.Update(key(tui.KeyEnter)) // root is a branch
	if cmd != nil {
		t.Error("Enter on a branch should toggle, not emit a Cmd")
	}
}

func TestCursorMovementAcrossFlattenedItems(t *testing.T) {
	m := testTree()
	next, _ := m.Update(key(tui.KeyEnter)) // expand root: 3 visible items
	m = next

	next, _ = m.Update(key(tui.KeyDown))
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
}

func TestEmptyTreeDoesNotPanic(t *testing.T) {
	m := New()
	m.SetCursor(3)
	if m.Cursor() != 0 {
		t.Errorf("SetCursor on empty tree = %d, want 0", m.Cursor())
	}
	next, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil || next.Cursor() != 0 {
		t.Error("Update on an empty tree should be a safe no-op")
	}
	if got := m.View(); got != "" {
		t.Errorf("View() on empty tree = %q, want empty", got)
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := testTree()
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.Cursor() != m.Cursor() {
		t.Error("a non-Key Msg should be a no-op")
	}
}

func TestView(t *testing.T) {
	dt := theme.DarkTheme()
	m := testTree()
	m.Theme = dt
	next, _ := m.Update(key(tui.KeyEnter)) // expand root
	m = next

	cursorStyle := dt.ResolvedStates().Selected.Bold()
	want := "> ▾ " + cursorStyle.Render("root") +
		"\n  " + "  " + "▸ child0" +
		"\n  " + "  " + "  child1"

	if got := m.View(); got != want {
		t.Errorf("View() =\n%q\nwant\n%q", got, want)
	}
}

func TestRightOnCollapsedBranchExpands(t *testing.T) {
	m := testTree()
	if m.IsExpanded("0") {
		t.Fatal("root should start collapsed")
	}
	next, cmd := m.Update(key(tui.KeyRight))
	if cmd != nil {
		t.Error("Right on a branch should not emit a Cmd")
	}
	if !next.IsExpanded("0") {
		t.Fatal("Right on a collapsed branch should expand it")
	}
}

func TestLeftOnExpandedBranchCollapses(t *testing.T) {
	m := testTree()
	m.toggle("0") // expand root
	next, cmd := m.Update(key(tui.KeyLeft))
	if cmd != nil {
		t.Error("Left on a branch should not emit a Cmd")
	}
	if next.IsExpanded("0") {
		t.Fatal("Left on an expanded branch should collapse it")
	}
}

func TestLeftOnAlreadyCollapsedBranchIsNoop(t *testing.T) {
	m := testTree()
	if m.IsExpanded("0") {
		t.Fatal("root should start collapsed")
	}
	next, cmd := m.Update(key(tui.KeyLeft))
	if cmd != nil {
		t.Error("Left on an already-collapsed branch should not emit a Cmd")
	}
	if next.IsExpanded("0") {
		t.Fatal("Left on an already-collapsed branch should remain collapsed")
	}
	if next.Cursor() != m.Cursor() {
		t.Error("Left on an already-collapsed branch should not move the cursor")
	}
}

func TestLeftRightOnLeafIsNoop(t *testing.T) {
	m := testTree()
	next, _ := m.Update(key(tui.KeyEnter)) // expand root
	m = next
	m.SetCursor(2) // "child1", a leaf

	next, cmd := m.Update(key(tui.KeyRight))
	if cmd != nil {
		t.Error("Right on a leaf should not emit a Cmd")
	}
	if next.Cursor() != m.Cursor() {
		t.Error("Right on a leaf should not move the cursor")
	}

	next, cmd = m.Update(key(tui.KeyLeft))
	if cmd != nil {
		t.Error("Left on a leaf should not emit a Cmd")
	}
	if next.Cursor() != m.Cursor() {
		t.Error("Left on a leaf should not move the cursor")
	}
}

func TestLeftRightDoNotAffectEnterSpaceBehavior(t *testing.T) {
	// Regression guard for #570-#572's additive requirement: Enter/Space's
	// existing toggle-if-branch / confirm-if-leaf behavior must be exactly
	// as before Left/Right were added.
	m := testTree()
	_, cmd := m.Update(key(tui.KeyEnter)) // root is a branch: toggles, no Cmd
	if cmd != nil {
		t.Error("Enter on a branch should still toggle, not emit a Cmd")
	}
	next, _ := m.Update(key(tui.KeySpace)) // toggles root again (still collapsed -> now expanded via Space)
	if !next.IsExpanded("0") {
		t.Fatal("Space on a collapsed branch should still expand it")
	}
}

func TestVisibleRowsMatchesFlatten(t *testing.T) {
	m := testTree()
	m.toggle("0") // expand root

	rows := m.VisibleRows()
	items := m.flatten()
	if len(rows) != len(items) {
		t.Fatalf("VisibleRows() len = %d, want %d", len(rows), len(items))
	}
	for i, item := range items {
		row := rows[i]
		if row.Label != item.node.Label || row.Path != item.path || row.Depth != item.depth {
			t.Errorf("VisibleRows()[%d] = %+v, want Label=%q Path=%q Depth=%d", i, row, item.node.Label, item.path, item.depth)
		}
		wantHasChildren := len(item.node.Children) > 0
		if row.HasChildren != wantHasChildren {
			t.Errorf("VisibleRows()[%d].HasChildren = %v, want %v", i, row.HasChildren, wantHasChildren)
		}
	}
}

func TestVisibleRowsEmptyTree(t *testing.T) {
	m := New()
	if rows := m.VisibleRows(); len(rows) != 0 {
		t.Errorf("VisibleRows() on an empty tree = %v, want none", rows)
	}
}
