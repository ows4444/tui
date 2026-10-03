package filepicker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ows4444/tui"
)

func setupTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "a.go"), nil, 0o644))
	must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "sub", "nested.go"), nil, 0o644))
	return dir
}

func TestExtensionFilter(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	m.Extensions = []string{".go"}
	m = m.Reload()

	var names []string
	for _, e := range m.Entries() {
		names = append(names, e.Name)
	}
	// dirs are always listed for nav, files are filtered by extension.
	want := []string{"sub", "a.go"}
	if len(names) != len(want) {
		t.Fatalf("Entries() = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("entry %d = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestDirsOnlyExcludesFiles(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	m.DirsOnly = true
	m = m.Reload()

	if len(m.Entries()) != 1 || m.Entries()[0].Name != "sub" {
		t.Fatalf("Entries() = %v, want just [sub]", m.Entries())
	}
}

func TestEnterOnFileEmitsSelectedMsgWithOnePath(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	// cursor 0 is "a.go" (dirs-first sort puts "sub" first, then files a.go, b.txt)
	idx := indexOf(m, "a.go")
	m.cursor = idx

	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on a file: want a Cmd delivering SelectedMsg, got nil")
	}
	msg, ok := cmd().(SelectedMsg)
	if !ok {
		t.Fatalf("Update returned Cmd delivering %T, want SelectedMsg", cmd())
	}
	if msg.Path != filepath.Join(dir, "a.go") {
		t.Errorf("SelectedMsg.Path = %q, want %q", msg.Path, filepath.Join(dir, "a.go"))
	}
}

func TestEnterOnDirDescendsNotSelects(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	m.cursor = indexOf(m, "sub")

	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd != nil {
		t.Fatal("Enter on a dir (not DirsOnly): want no Cmd, got one")
	}
	if next.Dir != filepath.Join(dir, "sub") {
		t.Errorf("Dir = %q, want descended into sub", next.Dir)
	}
}

func TestEnterOnDirInDirsOnlyModeSelects(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	m.DirsOnly = true
	m = m.Reload()
	m.cursor = 0 // only "sub" is listed

	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on a dir in DirsOnly mode: want a Cmd delivering SelectedMsg, got nil")
	}
	msg := cmd().(SelectedMsg)
	if msg.Path != filepath.Join(dir, "sub") {
		t.Errorf("SelectedMsg.Path = %q, want %q", msg.Path, filepath.Join(dir, "sub"))
	}
}

func TestLeftAscendsToParent(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	m.cursor = indexOf(m, "sub")
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	if m.Dir != filepath.Join(dir, "sub") {
		t.Fatalf("Right on dir: Dir = %q, want %q", m.Dir, filepath.Join(dir, "sub"))
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	if m.Dir != dir {
		t.Errorf("Left: Dir = %q, want back to %q", m.Dir, dir)
	}
}

func TestUnreadableDirDoesNotPanic(t *testing.T) {
	m := New(filepath.Join(t.TempDir(), "does-not-exist"))
	if len(m.Entries()) != 0 {
		t.Fatalf("Entries() = %v, want empty for an unreadable dir", m.Entries())
	}
	if view := m.View(); view != "" {
		t.Errorf("View() = %q, want empty", view)
	}
}

func TestCursorMovement(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	n := len(m.Entries())
	if n < 2 {
		t.Fatalf("test fixture too small: %d entries", n)
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.Cursor() != 1 {
		t.Errorf("Cursor() after Down = %d, want 1", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	if m.Cursor() != 0 {
		t.Errorf("Cursor() after Up = %d, want 0", m.Cursor())
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	dir := setupTree(t)
	m := New(dir)
	next, cmd := m.Update(struct{}{})
	if cmd != nil {
		t.Error("non-Key msg: want nil Cmd")
	}
	if next.Cursor() != m.Cursor() {
		t.Error("non-Key msg: want cursor unchanged")
	}
}

func indexOf(m Model, name string) int {
	for i, e := range m.Entries() {
		if e.Name == name {
			return i
		}
	}
	return -1
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
