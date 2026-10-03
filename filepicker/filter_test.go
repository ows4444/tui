package filepicker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/filepicker"
)

// filterTree is a directory with two files, one directory and a nested file.
func filterTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"a.go", "b.txt", filepath.Join("sub", "nested.go")} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func names(m filepicker.Model) string {
	var out []string
	for _, e := range m.Entries() {
		out = append(out, e.Name)
	}
	return strings.Join(out, " ")
}

// selectEach presses Enter on every listed entry and returns the paths that
// were confirmed.
func selectEach(m filepicker.Model) []string {
	var picked []string
	for i := range m.Entries() {
		at := m
		for j := 0; j < i; j++ {
			at, _ = at.Update(tui.Key{Type: tui.KeyDown})
		}
		if _, cmd := at.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
			if sel, ok := cmd().(filepicker.SelectedMsg); ok {
				picked = append(picked, filepath.Base(sel.Path))
			}
		}
	}
	return picked
}

// Extensions set after New and applied with Reload filters the starting
// directory, and a filtered file can be neither seen nor selected.
func TestExtensionsApplyToTheStartingDirectory(t *testing.T) {
	m := filepicker.New(filterTree(t))
	if got := names(m); got != "sub a.go b.txt" {
		t.Fatalf("unfiltered listing = %q", got)
	}
	m.Extensions = []string{".go"}
	m = m.Reload()
	if got := names(m); got != "sub a.go" {
		t.Fatalf("listing with Extensions = %q, want %q", got, "sub a.go")
	}
	if strings.Contains(m.View(), "b.txt") {
		t.Errorf("View shows a filtered file:\n%s", m.View())
	}
	if got := strings.Join(selectEach(m), " "); got != "a.go" {
		t.Errorf("selectable = %q, want only a.go", got)
	}
}

// DirsOnly set after New and applied with Reload lists only directories.
func TestDirsOnlyAppliesToTheStartingDirectory(t *testing.T) {
	m := filepicker.New(filterTree(t))
	m.DirsOnly = true
	m = m.Reload()
	if got := names(m); got != "sub" {
		t.Fatalf("listing with DirsOnly = %q, want %q", got, "sub")
	}
	if got := strings.Join(selectEach(m), " "); got != "sub" {
		t.Errorf("selectable = %q, want only sub", got)
	}
}

// A struct literal lists nothing until Reload.
func TestReloadListsAStructLiteral(t *testing.T) {
	m := filepicker.Model{Dir: filterTree(t), Extensions: []string{".txt"}}
	if len(m.Entries()) != 0 {
		t.Fatalf("a struct literal listed %q before Reload", names(m))
	}
	if got := names(m.Reload()); got != "sub b.txt" {
		t.Fatalf("after Reload = %q, want %q", got, "sub b.txt")
	}
}
