package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// tui.Terminal is the one terminal port. internal/termio holds implementations
// (the OS adapter and a fake) and must not declare a second interface for them
// to drift from: that is how the two copies this test replaces came about.
func TestSinglePortDefinition(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); path != root && (n == "testdata" || n == "examples" || n == "tools" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		dir, _ := filepath.Rel(root, filepath.Dir(path))
		isRoot := dir == "."
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				if _, isIface := ts.Type.(*ast.InterfaceType); isIface && ts.Name.Name == "Terminal" && !isRoot {
					t.Errorf("%s declares interface Terminal; tui.Terminal is the one port", filepath.ToSlash(path))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Every public package carries at least one runnable Example, so its godoc
// page shows how it is used and the usage is compiled (and, with an Output
// comment, checked) by go test.
func TestEveryPublicPackageHasExample(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	hasExample := map[string]bool{}
	seen := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); path != root && (n == "testdata" || n == "internal" || n == "examples" || n == "tools" || n == "scripts" || n == "docs" || n == "bench" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		seen[dir] = true
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Example") {
				hasExample[dir] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) < 50 {
		t.Fatalf("found only %d packages; is the test running from internal/archtest?", len(seen))
	}
	for dir := range seen {
		if !hasExample[dir] {
			t.Errorf("%s has no Example function", strings.TrimPrefix(dir, "../../"))
		}
	}
}

// ansi.SetClusterWidth is the process-wide default: an application may set it
// once, but no library code may, because it would change what every other
// Program in the process measures (the probe once did). Per-terminal widths
// travel as an ansi.Measurer instead.
func TestNoGlobalWidthToggle(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); path != root && (n == "testdata" || n == "ansi" || n == "archtest" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "SetClusterWidth" {
				t.Errorf("%s calls SetClusterWidth; carry an ansi.Measurer instead", filepath.ToSlash(path))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
