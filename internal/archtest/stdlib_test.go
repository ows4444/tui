package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Supply-chain guarantee: every package in the module (examples and tools
// included) imports only the Go standard library and this module itself, and
// go.mod requires nothing.

// isStdlib reports whether an import path belongs to the standard library.
// Standard library paths have no dot in their first element; third-party
// paths start with a domain name.
func isStdlib(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

// foreignImports returns "dir imports path" for every non-test Go file under
// root that imports a package outside the standard library and the module.
func foreignImports(t *testing.T, root string) []string {
	t.Helper()
	var bad []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); path != root && (n == "testdata" || n == "vendor" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		dir, _ := filepath.Rel(root, filepath.Dir(path))
		dir = filepath.ToSlash(dir)
		for _, im := range f.Imports {
			ip, err := strconv.Unquote(im.Path.Value)
			if err != nil {
				return err
			}
			if isStdlib(ip) || ip == mod || strings.HasPrefix(ip, mod+"/") {
				continue
			}
			bad = append(bad, "package "+dir+" imports non-standard-library package "+ip)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(bad)
	return bad
}

func TestLibraryImportsStdlibOnly(t *testing.T) {
	root := moduleRoot(t)
	for _, b := range foreignImports(t, root) {
		t.Error(b)
	}
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if s := strings.TrimSpace(line); strings.HasPrefix(s, "require") || strings.HasPrefix(s, "replace") {
			t.Errorf("go.mod must stay dependency-free, found %q", s)
		}
	}
}

func TestStdlibRuleCatchesForeignImport(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("good/good.go", "package good\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n\t\"github.com/ows4444/tui/ansi\"\n)\n")
	if got := foreignImports(t, dir); len(got) != 0 {
		t.Fatalf("clean fixture flagged: %v", got)
	}
	write("bad/bad.go", "package bad\n\nimport _ \"golang.org/x/sys/unix\"\n")
	write("bad2/bad2.go", "package bad2\n\nimport _ \"github.com/mattn/go-runewidth\"\n")
	got := foreignImports(t, dir)
	want := []string{
		"package bad imports non-standard-library package golang.org/x/sys/unix",
		"package bad2 imports non-standard-library package github.com/mattn/go-runewidth",
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("foreign imports = %v, want %v", got, want)
	}
}
