package treeview

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// lazyTree has one lazy root whose Load is counted.
func lazyTree(calls *atomic.Int32) Model {
	return New(Node{Label: "root", Load: func() []Node {
		calls.Add(1)
		return []Node{{Label: "a"}, {Label: "b", Load: func() []Node { return []Node{{Label: "deep"}} }}}
	}})
}

// run executes the Cmd and feeds its message back, as the program would.
func run(t *testing.T, m Model, cmd tui.Cmd) (Model, tui.Msg) {
	t.Helper()
	if cmd == nil {
		t.Fatal("no Cmd")
	}
	msg := cmd()
	m, next := m.Update(msg)
	if next != nil {
		return m, next()
	}
	return m, nil
}

func TestLazyNodeLoadsOnExpand(t *testing.T) {
	var calls atomic.Int32
	m := lazyTree(&calls)
	if rows := m.VisibleRows(); len(rows) != 1 || !rows[0].HasChildren {
		t.Fatalf("a lazy node must look like a branch: %+v", rows)
	}
	m, cmd := m.Update(key(tui.KeyRight))
	if calls.Load() != 0 {
		t.Fatal("Load ran inside Update; it must run in the Cmd")
	}
	if got := len(m.VisibleRows()); got != 1 {
		t.Fatalf("opened before the children arrived: %d rows", got)
	}
	m2, _ := m.Update(key(tui.KeyRight)) // while loading: no second Load
	m, _ = run(t, m2, cmd)
	if calls.Load() != 1 {
		t.Fatalf("Load ran %d times; want 1", calls.Load())
	}
	rows := m.VisibleRows()
	if len(rows) != 3 || rows[1].Label != "a" || rows[2].Label != "b" || !rows[2].HasChildren || rows[1].HasChildren {
		t.Fatalf("rows after load: %+v", rows)
	}
	if !m.IsExpanded("0") {
		t.Fatal("node did not open after its children arrived")
	}
	// Collapse and expand again: no second load.
	m.SetCursor(0)
	m, _ = m.Update(key(tui.KeyLeft))
	m, cmd = m.Update(key(tui.KeyRight))
	if cmd != nil || len(m.VisibleRows()) != 3 || calls.Load() != 1 {
		t.Fatalf("re-expanding loaded again: calls %d rows %d", calls.Load(), len(m.VisibleRows()))
	}
}

func TestLazySecondLevelAndSelect(t *testing.T) {
	var calls atomic.Int32
	m := lazyTree(&calls)
	m, cmd := m.Update(key(tui.KeyEnter))
	m, _ = run(t, m, cmd)
	m.SetCursor(2) // b
	m, cmd = m.Update(key(tui.KeyEnter))
	m, _ = run(t, m, cmd)
	if rows := m.VisibleRows(); len(rows) != 4 || rows[3].Label != "deep" || rows[3].Path != "0.1.0" {
		t.Fatalf("second level: %+v", rows)
	}
	m.SetCursor(3)
	_, cmd = m.Update(key(tui.KeyEnter))
	if sel, ok := cmd().(SelectedMsg); !ok || sel.Path != "0.1.0" || sel.Node.Label != "deep" {
		t.Fatalf("selecting a loaded leaf: %+v", sel)
	}
	if v := ansi.StripANSI(m.View()); !strings.Contains(v, "deep") {
		t.Fatalf("view lacks the loaded node:\n%s", v)
	}
}

func TestLazyNameKeepsTreesApart(t *testing.T) {
	var calls atomic.Int32
	a, b := lazyTree(&calls), lazyTree(&calls)
	a.Name, b.Name = "a", "b"
	a, cmd := a.Update(key(tui.KeyRight))
	b, _ = b.Update(key(tui.KeyRight))
	msg := cmd()
	b, _ = b.Update(msg)
	if len(b.VisibleRows()) != 1 {
		t.Fatal("tree b took tree a's LoadedMsg")
	}
	a, _ = a.Update(msg)
	if len(a.VisibleRows()) != 3 {
		t.Fatal("tree a lost its own LoadedMsg")
	}
	// A message nobody asked for is ignored.
	c := lazyTree(&calls)
	c, _ = c.Update(LoadedMsg{Path: "0", Children: []Node{{Label: "x"}}})
	if len(c.VisibleRows()) != 1 {
		t.Fatal("an unrequested LoadedMsg opened a node")
	}
}

func TestLazyEmptyLoad(t *testing.T) {
	m := New(Node{Label: "empty", Load: func() []Node { return nil }})
	m, cmd := m.Update(key(tui.KeyRight))
	m, _ = run(t, m, cmd)
	if len(m.VisibleRows()) != 1 || !m.IsExpanded("0") {
		t.Fatalf("empty load: rows %d expanded %v", len(m.VisibleRows()), m.IsExpanded("0"))
	}
}

func TestNotify(t *testing.T) {
	m := New(Node{Label: "r", Children: []Node{{Label: "c"}}})
	m, cmd := m.Update(key(tui.KeyRight))
	if cmd != nil {
		t.Fatal("Notify off must return no Cmd")
	}
	m.Notify = true
	m, _ = m.Update(key(tui.KeyLeft))
	_, cmd = m.Update(key(tui.KeyRight))
	if e, ok := cmd().(ExpandedMsg); !ok || e.Path != "0" || e.Node.Label != "r" {
		t.Fatalf("expand message: %+v", e)
	}
	m, _ = m.Update(key(tui.KeyRight))
	_, cmd = m.Update(key(tui.KeyLeft))
	if c, ok := cmd().(CollapsedMsg); !ok || c.Path != "0" {
		t.Fatalf("collapse message: %+v", c)
	}
}

func TestNotifyAfterLazyLoad(t *testing.T) {
	var calls atomic.Int32
	m := lazyTree(&calls)
	m.Notify = true
	m, cmd := m.Update(key(tui.KeyRight))
	_, got := run(t, m, cmd)
	if e, ok := got.(ExpandedMsg); !ok || e.Node == nil || e.Node.Label != "root" {
		t.Fatalf("after load: %+v", got)
	}
}

func TestLazyDoesNotMutateRoots(t *testing.T) {
	var calls atomic.Int32
	m := lazyTree(&calls)
	m2, cmd := m.Update(key(tui.KeyRight))
	m2, _ = run(t, m2, cmd)
	if len(m.Roots[0].Children) != 0 || len(m.VisibleRows()) != 1 || len(m2.VisibleRows()) != 3 {
		t.Fatal("loading changed the Roots or an earlier copy of the Model")
	}
}

func TestLazyFilterSearchesLoadedNodes(t *testing.T) {
	var calls atomic.Int32
	m := lazyTree(&calls)
	m.Filter = true
	m, cmd := m.Update(key(tui.KeyRight))
	m, _ = run(t, m, cmd)
	m.SetQuery("a")
	found := false
	for _, r := range m.VisibleRows() {
		found = found || r.Label == "a"
	}
	if !found {
		t.Fatalf("a loaded node is not searched: %+v", m.VisibleRows())
	}
}

func TestLazyDirectoryTree(t *testing.T) {
	dir := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(dir, "sub", "inner"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "sub", "f.txt"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "top.txt"), nil, 0o644))
	root := LazyDirectoryTree(dir)
	if len(root.Children) != 0 || root.Load == nil {
		t.Fatal("LazyDirectoryTree walked up front")
	}
	m := New(root)
	m, cmd := m.Update(key(tui.KeyRight))
	m, _ = run(t, m, cmd)
	rows := m.VisibleRows()
	if len(rows) != 3 || rows[1].Label != "sub" || !rows[1].HasChildren || rows[2].Label != "top.txt" || rows[2].HasChildren {
		t.Fatalf("top level: %+v", rows)
	}
	m.SetCursor(1)
	m, cmd = m.Update(key(tui.KeyRight))
	m, _ = run(t, m, cmd)
	if got := len(m.VisibleRows()); got != 5 {
		t.Fatalf("after opening sub: %d rows, want 5: %+v", got, m.VisibleRows())
	}
	if got := LazyDirectoryTree(filepath.Join(dir, "missing")).Load(); len(got) != 0 {
		t.Fatalf("a missing directory listed %v", got)
	}
}
