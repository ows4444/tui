package layout_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deprecatedLayoutFuncs returns the package-level functions in package layout
// whose doc comment has a "Deprecated:" paragraph.
func deprecatedLayoutFuncs(t *testing.T) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || fn.Doc == nil || !fn.Name.IsExported() {
					continue
				}
				if strings.Contains(fn.Doc.Text(), "Deprecated:") {
					out[fn.Name.Name] = true
				}
			}
		}
	}
	return out
}

// TestExamplesUseNoDeprecatedLayoutAPI proves criterion #44: the equivalent of
// a staticcheck SA1019 run over examples/ finds no deprecated layout function.
// It flags layout.<Func> for every deprecated package-level function, and
// Render called on a chain that starts at layout.NewBox() (the deprecated
// string path of Box); a Box held in a variable is not tracked.
func TestExamplesUseNoDeprecatedLayoutAPI(t *testing.T) {
	deprecated := deprecatedLayoutFuncs(t)
	fset := token.NewFileSet()
	err := filepath.WalkDir("../examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if id, ok := n.X.(*ast.Ident); ok && id.Name == "layout" && deprecated[n.Sel.Name] {
					t.Errorf("%s: layout.%s is deprecated", fset.Position(n.Pos()), n.Sel.Name)
				}
				if n.Sel.Name == "Render" && chainStartsAtNewBox(n.X) {
					t.Errorf("%s: Box.Render is deprecated; use layout.BoxNode", fset.Position(n.Pos()))
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// chainStartsAtNewBox reports whether e is a method chain rooted at a call to
// layout.NewBox.
func chainStartsAtNewBox(e ast.Expr) bool {
	for {
		call, ok := e.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			return id.Name == "layout" && sel.Sel.Name == "NewBox"
		}
		e = sel.X
	}
}

// TestRemovedStringLayoutAPIIsGone proves the first half of criterion #104:
// the string join helpers removed for v1.0 are no longer declared, exported,
// anywhere in package layout.
func TestRemovedStringLayoutAPIIsGone(t *testing.T) {
	removed := map[string]bool{
		"JoinHorizontal": true, "JoinHorizontalAlign": true, "JoinVertical": true, "JoinVerticalAlign": true,
		"FlexRow": true, "FlexItem": true, "GridFlex": true, "ColSpec": true, "Grid": true,
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				switch d := d.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil && removed[d.Name.Name] {
						t.Errorf("layout.%s is declared; it was removed for v1.0", d.Name.Name)
					}
				case *ast.GenDecl:
					for _, sp := range d.Specs {
						if ts, ok := sp.(*ast.TypeSpec); ok && removed[ts.Name.Name] {
							t.Errorf("layout.%s is declared; it was removed for v1.0", ts.Name.Name)
						}
					}
				}
			}
		}
	}
}

// TestMigrationGuideMapsEachRemovedIdentifier proves the second half of
// criterion #104: docs/migrating-to-v1.md has a section for each removed
// identifier and the section names the node that replaces it.
func TestMigrationGuideMapsEachRemovedIdentifier(t *testing.T) {
	guide, err := os.ReadFile("../docs/migrating-to-v1.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(guide)
	replacement := map[string]string{
		"JoinHorizontal": "Row", "JoinHorizontalAlign": "CrossAlign", "JoinVertical": "Column",
		"JoinVerticalAlign": "CrossAlign", "FlexRow": "Row", "GridFlex": "GridNode", "Grid": "GridNode",
		"ColSpec": "Track", "FlexItem": "FlexChild",
	}
	for name, node := range replacement {
		heading := "### " + name + "\n"
		i := strings.Index(text, heading)
		if i < 0 {
			t.Errorf("the migration guide has no section %q", "### "+name)
			continue
		}
		section := text[i+len(heading):]
		if j := strings.Index(section, "\n### "); j >= 0 {
			section = section[:j]
		}
		if !strings.Contains(section, node) {
			t.Errorf("the section for %s does not name its replacement %s", name, node)
		}
	}
}
