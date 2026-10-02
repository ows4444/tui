package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// layoutAllowlist names the library packages that define a Model but no
// LayoutNode method, each with the reason none is needed. A package with a
// Model must either define a LayoutNode method or be listed here.
var layoutAllowlist = map[string]string{}

// scanLayoutPackages walks the library packages under root and returns, for
// every package that declares a type named Model, whether the package also
// declares a LayoutNode method.
func scanLayoutPackages(t *testing.T, root string) map[string]bool {
	t.Helper()
	hasModel := map[string]bool{}
	hasNode := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch name := d.Name(); {
			case path == root:
			case name == "examples" || name == "tools" || name == "internal" || name == "testdata" || strings.HasPrefix(name, "."):
				return filepath.SkipDir
			}
			return nil
		}
		dir := filepath.Dir(path)
		if dir == root || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		pkg, _ := filepath.Rel(root, dir)
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.Name == "Model" {
						hasModel[pkg] = true
					}
				}
			case *ast.FuncDecl:
				if d.Recv != nil && d.Name.Name == "LayoutNode" {
					hasNode[pkg] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for pkg := range hasModel {
		out[pkg] = hasNode[pkg]
	}
	return out
}

// missingLayoutNode lists the packages with a Model and neither a LayoutNode
// nor an allowlist entry, sorted.
func missingLayoutNode(t *testing.T, root string, allow map[string]string) []string {
	t.Helper()
	var bad []string
	for pkg, ok := range scanLayoutPackages(t, root) {
		if !ok && allow[pkg] == "" {
			bad = append(bad, pkg)
		}
	}
	sort.Strings(bad)
	return bad
}

func TestEveryModelHasLayoutNodeOrIsAllowlisted(t *testing.T) {
	if bad := missingLayoutNode(t, ".", layoutAllowlist); len(bad) > 0 {
		t.Fatalf("these packages define a Model with no LayoutNode method and no allowlist entry: %s", strings.Join(bad, ", "))
	}
}

// Every allowlist entry is still needed: a package that gained a LayoutNode or
// lost its Model must be taken off the list.
func TestLayoutAllowlistHasNoStaleEntries(t *testing.T) {
	seen := scanLayoutPackages(t, ".")
	for pkg, why := range layoutAllowlist {
		hasNode, hasModel := seen[pkg]
		switch {
		case !hasModel:
			t.Errorf("allowlisted package %q has no Model", pkg)
		case hasNode:
			t.Errorf("allowlisted package %q has a LayoutNode now; remove it from the list", pkg)
		case strings.TrimSpace(why) == "":
			t.Errorf("allowlisted package %q has no reason", pkg)
		}
	}
}

// The rule bites: on a scratch tree, a Model with no LayoutNode fails naming
// its package, adding a LayoutNode or an allowlist entry makes it pass, and
// removing the LayoutNode again makes it fail.
func TestLayoutConformanceFailsWhenLayoutNodeIsRemoved(t *testing.T) {
	root := t.TempDir()
	write := func(src string) {
		t.Helper()
		dir := filepath.Join(root, "fancy")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "fancy.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const without = "package fancy\n\ntype Model struct{}\n"
	const with = without + "\nfunc (m Model) LayoutNode() int { return 0 }\n"

	write(without)
	if bad := missingLayoutNode(t, root, nil); len(bad) != 1 || bad[0] != "fancy" {
		t.Fatalf("Model without LayoutNode: got %v, want [fancy]", bad)
	}
	if bad := missingLayoutNode(t, root, map[string]string{"fancy": "a reason"}); len(bad) != 0 {
		t.Errorf("allowlisted package still reported: %v", bad)
	}
	write(with)
	if bad := missingLayoutNode(t, root, nil); len(bad) != 0 {
		t.Errorf("package with LayoutNode reported: %v", bad)
	}
	write(without)
	if bad := missingLayoutNode(t, root, nil); len(bad) != 1 {
		t.Errorf("removing the LayoutNode did not make the check fail: %v", bad)
	}
}
