package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// widgetMethods parses the non-test files of every top-level package that
// declares a Model with Update and View methods (a widget) and returns, per
// package, the set of method names declared on any receiver.
func widgetMethods(t *testing.T) map[string]map[string]bool {
	t.Helper()
	dirs, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]bool{}
	for _, d := range dirs {
		if !d.IsDir() || d.Name() == "internal" || d.Name() == "examples" || d.Name() == "tools" ||
			d.Name() == "tuitest" /* a test harness, not a widget */ {
			continue
		}
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, d.Name(), func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatal(err)
		}
		methods := map[string]bool{}
		for _, p := range pkgs {
			for _, f := range p.Files {
				for _, decl := range f.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
						methods[fn.Name.Name] = true
					}
				}
			}
		}
		if methods["Update"] && methods["View"] {
			out[filepath.Base(d.Name())] = methods
		}
	}
	return out
}

// TestREADMECapabilityClaimsHold fails, naming the package, when README says
// every widget has LayoutNode or Linearize and a widget package lacks it.
func TestREADMECapabilityClaimsHold(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	claims := map[string]string{
		"LayoutNode": "Every widget has a `LayoutNode()` adapter",
		"Linearize":  "every widget implements `Linearize`",
	}
	widgets := widgetMethods(t)
	if len(widgets) == 0 {
		t.Fatal("found no widget packages")
	}
	for method, claim := range claims {
		if !strings.Contains(text, claim) {
			t.Errorf("README no longer contains %q; update this test with it", claim)
			continue
		}
		var missing []string
		for pkg, methods := range widgets {
			if !methods[method] {
				missing = append(missing, pkg)
			}
		}
		sort.Strings(missing)
		for _, pkg := range missing {
			t.Errorf("README claims every widget has %s, but package %s lacks it", method, pkg)
		}
	}
}
