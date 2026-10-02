package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// mouseAllowlist names the widget packages that handle a key for navigation
// but not the mouse, each with the reason. The test fails if a package is on
// the list and does handle the mouse (prune it), or is not a widget package.
var mouseAllowlist = map[string]string{}

// TestKeyNavigationPackagesHandleMouse enforces acceptance criterion #75: the
// system shall handle MouseEvent in every package that handles Key for
// navigation. A widget package is one whose non-test sources declare an Update
// method. It handles Key when it names tui.Key or input.Key or calls
// keymap.Matches, and handles the mouse when it names MouseEvent or MouseMsg
// (tui.MouseEvent, input.MouseEvent, ...) anywhere in its non-test sources.
func TestKeyNavigationPackagesHandleMouse(t *testing.T) {
	root := filepath.Join("..", "..")
	type info struct{ update, key, mouse bool }
	pkgs := map[string]*info{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			switch {
			case rel == ".":
				return nil
			case rel == "internal" || rel == "examples" || rel == "docs" || rel == "testdata" || rel == "skeleton",
				strings.HasPrefix(rel, "internal/"), strings.HasPrefix(rel, "examples/"),
				strings.HasPrefix(rel, ".") || strings.HasSuffix(rel, "/testdata"):
				return filepath.SkipDir
			}
			return nil
		}
		dir := filepath.Dir(rel)
		if dir == "." || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		in := pkgs[dir]
		if in == nil {
			in = &info{}
			pkgs[dir] = in
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				if x.Recv != nil && x.Name.Name == "Update" {
					in.update = true
				}
			case *ast.SelectorExpr:
				pkg, ok := x.X.(*ast.Ident)
				if !ok {
					break
				}
				switch {
				case (pkg.Name == "tui" || pkg.Name == "input") && x.Sel.Name == "Key",
					pkg.Name == "keymap" && x.Sel.Name == "Matches":
					in.key = true
				case strings.HasPrefix(x.Sel.Name, "MouseEvent") || x.Sel.Name == "MouseMsg":
					in.mouse = true
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for p := range pkgs {
		names = append(names, p)
	}
	slices.Sort(names)
	for _, p := range names {
		in := pkgs[p]
		reason, listed := mouseAllowlist[p]
		switch {
		case in.update && in.key && !in.mouse && !listed:
			t.Errorf("%s handles Key in Update but never references MouseEvent: handle the mouse or add it to mouseAllowlist with a reason", p)
		case listed && (!in.update || !in.key || in.mouse):
			t.Errorf("%s is on mouseAllowlist (%q) but no longer needs to be: remove it", p, reason)
		}
	}
}
