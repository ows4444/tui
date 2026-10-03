package filepicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func TestSetThemeReplacesTheme(t *testing.T) {
	m := New(t.TempDir()).SetTheme(theme.LightTheme())
	if m.Theme != theme.LightTheme() {
		t.Fatalf("Theme = %+v, want theme.Light", m.Theme)
	}
}

func TestUnreadableDirListsNothing(t *testing.T) {
	m := New(filepath.Join(t.TempDir(), "missing"))
	if len(m.Entries()) != 0 || m.Cursor() != 0 {
		t.Fatalf("entries=%v cursor=%d, want empty at 0", m.Entries(), m.Cursor())
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
		t.Error("Enter on an empty listing returned a Cmd")
	}
	if got := m.View(); !strings.HasPrefix(got, "cannot read this directory: ") || m.Err() == nil {
		t.Errorf("View = %q, Err = %v; want the directory reported as unreadable", got, m.Err())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // descend with no entries
	if m.Dir == "" {
		t.Error("descend on an empty listing changed Dir")
	}
}

func TestFilterThatHidesEverythingResetsCursor(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644))
	m := New(dir)
	m.Extensions = []string{".go"}
	m = m.Reload()
	if len(m.Entries()) != 0 || m.Cursor() != 0 {
		t.Fatalf("entries=%v cursor=%d, want none at 0", m.Entries(), m.Cursor())
	}
}

func TestReloadClampsCursorWhenListShrinks(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a", "b", "c"} {
		must(t, os.WriteFile(filepath.Join(dir, n), nil, 0o644))
	}
	m := New(dir)
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	must(t, os.Remove(filepath.Join(dir, "b")))
	must(t, os.Remove(filepath.Join(dir, "c")))
	m = m.Reload()
	if m.Cursor() != 0 {
		t.Fatalf("Cursor = %d, want clamped to 0", m.Cursor())
	}
}

func TestRightOnFileDoesNotDescend(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644))
	m, _ := New(dir).Update(tui.Key{Type: tui.KeyRight})
	if m.Dir != dir {
		t.Fatalf("Dir = %q, want unchanged %q", m.Dir, dir)
	}
}

func TestLeftAtFilesystemRootStays(t *testing.T) {
	root := string(filepath.Separator)
	m, _ := New(root).Update(tui.Key{Type: tui.KeyLeft})
	if m.Dir != root {
		t.Fatalf("Dir = %q, want %q", m.Dir, root)
	}
}

func TestCursorStaysWithinBounds(t *testing.T) {
	m := New(setupTree(t))
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	if m.Cursor() != 0 {
		t.Fatalf("Up at top: Cursor = %d", m.Cursor())
	}
	for range 10 {
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	if m.Cursor() != len(m.Entries())-1 {
		t.Fatalf("Down at bottom: Cursor = %d, want %d", m.Cursor(), len(m.Entries())-1)
	}
	if _, cmd := m.Update(struct{}{}); cmd != nil {
		t.Error("non-key Msg returned a Cmd")
	}
}

func TestClamp(t *testing.T) {
	for _, c := range [][4]int{{-1, 0, 3, 0}, {2, 0, 3, 2}, {9, 0, 3, 3}} {
		if got := clamp(c[0], c[1], c[2]); got != c[3] {
			t.Errorf("clamp(%d,%d,%d) = %d, want %d", c[0], c[1], c[2], got, c[3])
		}
	}
}

func TestRenderZeroSizeAndEmptyListing(t *testing.T) {
	m := New(setupTree(t))
	if out := m.LayoutNode().Render(layout.Size{W: 0, H: 3}); out != "" {
		t.Errorf("zero width = %q, want empty", out)
	}
	if out := m.LayoutNode().Render(layout.Size{W: 10, H: 0}); out != "" {
		t.Errorf("zero height = %q, want empty", out)
	}
	empty := New(filepath.Join(t.TempDir(), "missing"))
	exact(t, empty.LayoutNode().Render(layout.Size{W: 6, H: 2}), 6, 2)
}
