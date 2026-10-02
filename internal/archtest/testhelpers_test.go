package archtest

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// testSupport are the packages that exist only to serve tests. A package
// outside _test.go files that imports one would put it in every build of the
// library.
var testSupport = []string{
	"github.com/ows4444/tui/internal/cellcheck",
	"github.com/ows4444/tui/internal/testutil",
}

// testSupportImports returns "file: imports path" for each non-test file under
// root that imports a test-support package. The test-support packages' own
// files and testdata are skipped.
func testSupportImports(root string) ([]string, error) {
	var out []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "testdata" || n == "vendor" || (path != root && strings.HasPrefix(n, ".")) {
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
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			for _, banned := range testSupport {
				if p == banned || strings.HasPrefix(p, banned+"/") {
					out = append(out, fmt.Sprintf("%s: imports %s", path, p))
				}
			}
		}
		return nil
	})
	return out, err
}

// No non-test file may import internal/cellcheck or internal/testutil, so the
// helpers cannot reach a production build.
func TestTestHelpersStayOutOfProductionCode(t *testing.T) {
	got, err := testSupportImports("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range got {
		t.Error(g)
	}
}

func TestTestHelperRuleCatchesViolations(t *testing.T) {
	root := t.TempDir()
	write := func(rel, src string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lib/lib.go", "package lib\n\nimport _ \"github.com/ows4444/tui/internal/cellcheck\"\n")
	write("lib/ok_test.go", "package lib\n\nimport _ \"github.com/ows4444/tui/internal/testutil\"\n")
	write("lib2/lib2.go", "package lib2\n\nimport _ \"github.com/ows4444/tui/internal/testutil\"\n")
	write("fine/fine.go", "package fine\n\nimport _ \"strings\"\n")

	got, err := testSupportImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 violations (lib.go, lib2.go), got %v", got)
	}
}
