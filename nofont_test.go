package tui

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The library must render on terminals without a patched Nerd Font, so no
// non-test source may embed a Unicode private-use glyph (where Nerd Font
// icons live). Callers may still pass such icons in themselves, e.g. via
// treeview.Sidebar's icons map.
func TestNoNerdFontGlyphsInLibrarySources(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			for _, r := range line {
				if (r >= 0xE000 && r <= 0xF8FF) || (r >= 0xF0000 && r <= 0x10FFFD) {
					t.Errorf("%s:%d: private-use glyph U+%X (Nerd Font) in library source", path, i+1, r)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
