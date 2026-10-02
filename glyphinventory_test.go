package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// glyphSources is the inventory of every non-test source file in the public
// packages that contains a non-ASCII string or rune literal, each with the
// reason it may. Every entry is a definition: a glyph set, a default or a font
// that the theme replaces at render time, so nothing here reaches the screen
// under theme.Dark.ASCII() (the rendered-output check for the whole widget
// set is the ASCII snapshot test). A widget that draws a fixed Unicode symbol
// instead of taking it from theme.Glyphs would add a file that is not listed
// here, and TestGlyphInventoryIsComplete fails until it is routed.
//
// The scan reads literals, so it cannot see glyphs computed at run time (faces
// and the charts build braille from code points); the rendered-output test in
// glyphrender_test.go covers those.
var glyphSources = map[string]string{
	"theme/glyphs.go":                "the Unicode and ASCII glyph sets themselves",
	"layout/box.go":                  "the Border sets, including ASCIIBorder; Theme.Border selects one",
	"spinner/spinner.go":             "the default frames, replaced by Theme.Glyphs.Spinner",
	"widgets/bigtext.go":             "the block font, drawn with full blocks and swapped for Theme.Glyphs.BarFull at render",
	"maskedinput/maskedinput.go":     "defaultMask, the documented default of the Mask field; it follows Theme.Glyphs.Mask",
	"passwordinput/passwordinput.go": "the exported Mask constant, the documented default; drawn from Theme.Glyphs.Mask",
}

// nonASCIIFiles returns the non-test source files of the public packages that
// hold a non-ASCII string or rune literal, keyed by slash path from the root.
func nonASCIIFiles(t *testing.T) map[string]int {
	t.Helper()
	found := map[string]int{}
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "examples", "tools", "internal", "testdata", "docs", "bench":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") || filepath.Dir(p) == "." {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || (bl.Kind != token.STRING && bl.Kind != token.CHAR) {
				return true
			}
			v, err := strconv.Unquote(bl.Value)
			if err != nil {
				v = bl.Value
			}
			for _, r := range v {
				if r >= 0x80 {
					found[filepath.ToSlash(p)]++
					break
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestGlyphInventoryIsComplete(t *testing.T) {
	found := nonASCIIFiles(t)

	var unlisted, stale []string
	for file := range found {
		if _, ok := glyphSources[file]; !ok {
			unlisted = append(unlisted, file)
		}
	}
	for file := range glyphSources {
		if _, ok := found[file]; !ok {
			stale = append(stale, file)
		}
	}
	sort.Strings(unlisted)
	sort.Strings(stale)
	for _, f := range unlisted {
		t.Errorf("%s has a non-ASCII literal but is not in glyphSources: draw it from theme.Glyphs (or ASCII), or list it here with a reason", f)
	}
	for _, f := range stale {
		t.Errorf("%s is in glyphSources but has no non-ASCII literal any more: remove it from the table", f)
	}

	t.Logf("%d files hold glyph definitions", len(glyphSources))
}
