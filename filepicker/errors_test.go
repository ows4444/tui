package filepicker_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/layout"
)

func enter(m filepicker.Model, name string) (filepicker.Model, tui.Msg) {
	for i, e := range m.Entries() {
		if e.Name != name {
			continue
		}
		for j := 0; j < i; j++ {
			m, _ = m.Update(tui.Key{Type: tui.KeyDown})
		}
		next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
		if cmd != nil {
			return next, cmd()
		}
		return next, nil
	}
	return m, "no entry " + name
}

// A directory that cannot be read is not shown as an empty one: Err reports
// why and View says so.
func TestUnreadableDirectoryIsReported(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("needs a directory the test cannot read")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if err := os.Mkdir(filepath.Join(dir, "empty"), 0o700); err != nil {
		t.Fatal(err)
	}

	m := filepicker.New(dir)
	if m.Err() != nil {
		t.Fatalf("Err on a readable directory = %v", m.Err())
	}
	in, _ := enter(m, "locked")
	if in.Dir != locked {
		t.Fatalf("Dir = %q, want %q", in.Dir, locked)
	}
	if !os.IsPermission(in.Err()) {
		t.Errorf("Err = %v, want a permission error", in.Err())
	}
	if v := in.View(); !strings.Contains(v, "cannot read") {
		t.Errorf("View of an unreadable directory = %q, want it to say so", v)
	}

	empty, _ := enter(m, "empty")
	if empty.Err() != nil || empty.View() != "" {
		t.Errorf("an empty directory: Err = %v, View = %q", empty.Err(), empty.View())
	}
	// Going back to a readable directory clears the error.
	back, _ := in.Update(tui.Key{Type: tui.KeyLeft})
	if back.Err() != nil || back.Dir != dir {
		t.Errorf("after going back: Dir = %q, Err = %v", back.Dir, back.Err())
	}
}

// A symlink to a directory is a directory: Enter descends into it. A broken
// symlink is listed as a file and does not fail the listing.
func TestSymlinks(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "inside.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "zlink")); err != nil {
		t.Skipf("cannot create symlinks here: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "broken")); err != nil {
		t.Fatal(err)
	}

	m := filepicker.New(dir)
	if m.Err() != nil {
		t.Fatalf("listing failed: %v", m.Err())
	}
	isDir := map[string]bool{}
	for _, e := range m.Entries() {
		isDir[e.Name] = e.IsDir
	}
	if len(isDir) != 3 || !isDir["real"] || !isDir["zlink"] || isDir["broken"] {
		t.Fatalf("entries (name: is a directory) = %v, want real and zlink as directories and broken as a file", isDir)
	}

	in, msg := enter(m, "zlink")
	if msg != nil {
		t.Fatalf("Enter on a symlinked directory produced %#v, want no Msg", msg)
	}
	if in.Dir != filepath.Join(dir, "zlink") || len(in.Entries()) != 1 || in.Entries()[0].Name != "inside.txt" {
		t.Fatalf("after Enter: Dir = %q, entries = %v", in.Dir, in.Entries())
	}
}

// The layout node and the accessible text report an unreadable directory too.
func TestUnreadableDirectoryInNodeAndLinearize(t *testing.T) {
	m := filepicker.New(filepath.Join(t.TempDir(), "missing"))
	if m.Err() == nil {
		t.Fatal("a missing directory has no Err")
	}
	if got := m.LayoutNode().Render(layout.Size{W: 60, H: 3}); !strings.Contains(got, "cannot read this directory") {
		t.Errorf("LayoutNode renders %q", got)
	}
	if got := m.Linearize(); !strings.HasSuffix(got, "file picker, cannot be read") {
		t.Errorf("Linearize = %q", got)
	}
}
