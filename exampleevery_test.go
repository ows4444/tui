package tui_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var exampleFunc = regexp.MustCompile(`(?m)^func Example`)

// TestEveryPackageHasExample fails when a non-internal package has no
// runnable Example function in its _test.go files.
func TestEveryPackageHasExample(t *testing.T) {
	type pkg struct{ hasGo, hasExample bool }
	pkgs := map[string]*pkg{}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			n := d.Name()
			if path != "." && (n == "internal" || n == "testdata" || n == "docs" || n == "bench" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		dir := filepath.Dir(path)
		p := pkgs[dir]
		if p == nil {
			p = &pkg{}
			pkgs[dir] = p
		}
		p.hasGo = true
		if strings.HasSuffix(path, "_test.go") {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if exampleFunc.Match(b) {
				p.hasExample = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) < 80 {
		t.Fatalf("only found %d packages; walk is broken", len(pkgs))
	}
	for dir, p := range pkgs {
		if p.hasGo && !p.hasExample {
			t.Errorf("package %s has no Example function", dir)
		}
	}
}
