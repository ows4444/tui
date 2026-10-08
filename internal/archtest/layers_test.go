package archtest

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const mod = "github.com/ows4444/tui"

// componentTier is the tier of every package not listed in tiers (and not under
// examples/ or internal/tools/): the stateful component packages.
const componentTier = 5

// tiers maps a module-relative import path ("" is the root runtime) to its tier.
var tiers = map[string]int{
	"ansi": 0, "layout": 0, "theme": 0, "motion": 0, "term": 0,
	"input": 1, "keymap": 1,
	"hittest": 2, "cellbuf": 2,
	"":        3,
	"widgets": 4, "widgets/chart": 4, "focus": 4, "markdown": 4, "tuitest": 4,
}

// internalTiers is the tier of every package under internal/ that is not a
// tool or this test. An internal package obeys the same downward rule as a
// public one; TestEveryInternalPackageHasATier fails for a package listed
// nowhere, so none defaults to a tier by accident.
var internalTiers = map[string]int{
	"internal/basetypes": 0, "internal/a11y": 0, "internal/bidi": 0, "internal/fsutil": 0,
	"internal/highlight": 0, "internal/ptytest": 0, "internal/boxdraw": 0,
	"internal/render": 2, "internal/termio": 2, "internal/capprobe": 2, "internal/announce": 2, "internal/braille": 2,
	"internal/cancelreader": 2, "internal/edit": 2, "internal/vtscreen": 2,
	// test support: import the root runtime, so no library package may import them
	"internal/cellcheck": 4, "internal/testutil": 4,
	// sweeps every component in its tests
	"internal/isolation": componentTier,
}

// composition lists the component-to-component imports that are allowed.
var composition = map[string][]string{
	"menu":           {"picker"},
	"menubar":        {"contextmenu"},
	"logview":        {"viewport"},
	"form":           {"textinput", "passwordinput"},
	"appshell":       {"textinput", "viewport"},
	"emailinput":     {"textinput"},
	"numberinput":    {"textinput"},
	"maskedinput":    {"textinput"},
	"passwordinput":  {"textinput"},
	"autocomplete":   {"textinput"},
	"buttongroup":    {"button"},
	"colorpicker":    {"textinput"},
	"commandpalette": {"textinput"},
	"taginput":       {"textinput"},
}

func rel(p string) string { return strings.TrimPrefix(strings.TrimPrefix(p, mod), "/") }

func tierOf(p string) int {
	if n, ok := tiers[p]; ok {
		return n
	}
	if n, ok := internalTiers[p]; ok {
		return n
	}
	return componentTier
}

func name(p string) string {
	if p == "" {
		return "tui"
	}
	return p
}

func skipped(p string) bool {
	return strings.HasPrefix(p, "examples/") || strings.HasPrefix(p, "internal/tools/") || p == "internal/archtest"
}

// imports returns, per module-relative package directory, the module-internal
// packages its non-test files import. It parses the files itself rather than
// running go list so that `go test` caching notices every file that changes.
func imports(t *testing.T, root string) map[string]map[string]bool {
	t.Helper()
	pkgs := map[string]map[string]bool{}
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
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		dir = filepath.ToSlash(dir)
		if dir == "." {
			dir = ""
		}
		if pkgs[dir] == nil {
			pkgs[dir] = map[string]bool{}
		}
		for _, im := range f.Imports {
			ip, err := strconv.Unquote(im.Path.Value)
			if err != nil {
				return err
			}
			if ip == mod || strings.HasPrefix(ip, mod+"/") {
				pkgs[dir][rel(ip)] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkgs
}

// Only the terminal adapter may import term: everything else reaches the
// terminal through internal/termio.
func TestOnlyTermioImportsTerm(t *testing.T) {
	for from, tos := range imports(t, "../..") {
		if tos["term"] && from != "internal/termio" && from != "term" && !skipped(from) {
			t.Errorf("%s imports term; use internal/termio", name(from))
		}
	}
}

// directionViolations returns one line for each import in pkgs that points up
// the tiers, naming both packages.
func directionViolations(pkgs map[string]map[string]bool) []string {
	var out []string
	for from, tos := range pkgs {
		if skipped(from) {
			continue
		}
		tf := tierOf(from)
		for to := range tos {
			tt := tierOf(to)
			switch {
			case tt < tf:
				continue
			case tt == tf && tf != componentTier:
				continue
			case tf == componentTier && tt == componentTier && slices.Contains(composition[from], to):
				continue
			}
			out = append(out, fmt.Sprintf("%s (tier %d) must not import %s (tier %d)", name(from), tf, name(to), tt))
		}
	}
	slices.Sort(out)
	return out
}

func TestImportDirection(t *testing.T) {
	pkgs := imports(t, "../..")
	if len(pkgs) < 50 {
		t.Fatalf("found only %d packages; is the test running from internal/archtest?", len(pkgs))
	}
	for _, v := range directionViolations(pkgs) {
		t.Error(v)
	}
}

func TestImportDirectionNamesBothPackagesOfAnUpwardImport(t *testing.T) {
	got := directionViolations(map[string]map[string]bool{
		"ansi":              {"internal/render": true},
		"internal/render":   {"ansi": true},
		"internal/vtscreen": {"internal/cellcheck": true},
	})
	want := []string{
		"ansi (tier 0) must not import internal/render (tier 2)",
		"internal/vtscreen (tier 2) must not import internal/cellcheck (tier 4)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// internalPackages returns the module-relative path of every directory under
// root/internal that holds a non-test Go file, excluding the tools.
func internalPackages(root string) ([]string, error) {
	seen := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == "testdata" || strings.HasPrefix(n, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			dir, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return err
			}
			if p := filepath.ToSlash(dir); !skipped(p) {
				seen[p] = true
			}
		}
		return nil
	})
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	slices.Sort(out)
	return out, err
}

// missingTiers returns the packages in pkgs with no entry in tiers or
// internalTiers.
func missingTiers(pkgs []string) []string {
	var out []string
	for _, p := range pkgs {
		_, a := tiers[p]
		_, b := internalTiers[p]
		if !a && !b {
			out = append(out, p)
		}
	}
	return out
}

func TestEveryInternalPackageHasATier(t *testing.T) {
	pkgs, err := internalPackages("../..")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) < 10 {
		t.Fatalf("found only %d internal packages; is the test running from internal/archtest?", len(pkgs))
	}
	for _, p := range missingTiers(pkgs) {
		t.Errorf("%s has no tier: add it to internalTiers in layers_test.go", p)
	}
	for p := range internalTiers {
		if !slices.Contains(pkgs, p) {
			t.Errorf("internalTiers lists %s, which does not exist", p)
		}
	}
}

func TestMissingTiersNamesThePackage(t *testing.T) {
	got := missingTiers([]string{"internal/render", "internal/brandnew"})
	if !slices.Equal(got, []string{"internal/brandnew"}) {
		t.Errorf("got %q, want [internal/brandnew]", got)
	}
}

// There are none to allow: a package variable any importer can reassign changes
// what every user of the package draws (the class of problem the theme presets
// were removed for). Export a function that returns a copy instead.
func TestNoExportedMutableVars(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); path != root && (n == "testdata" || n == "internal" || n == "examples" || n == "tools" || strings.HasPrefix(n, ".")) {
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
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				for _, n := range spec.(*ast.ValueSpec).Names {
					if !n.IsExported() || strings.HasPrefix(n.Name, "Err") {
						continue
					}
					t.Errorf("%s.%s: exported package variable; export a function that returns it instead", f.Name.Name, n.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// maxFileLines is the size past which a hand-written source file is a sign a
// second concern has crept in (program.go reached 1,897 lines before it was
// split). Generated files, tests, examples and tools are exempt.
const maxFileLines = 800

// oversize lists the files allowed past the limit, each with the reason.
var oversize = map[string]string{
	"textarea/textarea.go": "one type with one concern (an editor); splitting it would only scatter the cursor and wrap logic",
}

func TestFileSize(t *testing.T) {
	root := "../.."
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
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(src[:min(len(src), 512)], []byte("Code generated")) {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if n := bytes.Count(src, []byte("\n")); n > maxFileLines {
			if _, ok := oversize[rel]; !ok {
				t.Errorf("%s has %d lines (limit %d): split it by concern, or list it in oversize with a reason", rel, n, maxFileLines)
			}
		} else if reason, ok := oversize[rel]; ok {
			t.Errorf("%s is under the limit now; remove it from oversize (%s)", rel, reason)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Components are configured through exported fields on their value-typed
// Model; functional options belong to the runtime (tui.ProgramOption) and to
// builders below the component tier. A component package declaring its own
// functional-option type would be a third style.
func TestComponentsHaveNoFunctionalOptions(t *testing.T) {
	root := "../.."
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); path != root && (n == "testdata" || n == "internal" || n == "examples" || n == "tools" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dir, _ := filepath.Rel(root, filepath.Dir(path))
		if tierOf(filepath.ToSlash(dir)) != componentTier || dir == "." {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				if _, isFunc := ts.Type.(*ast.FuncType); isFunc && ts.Name.IsExported() &&
					(strings.HasSuffix(ts.Name.Name, "Option") || strings.HasSuffix(ts.Name.Name, "Opt")) {
					t.Errorf("%s.%s: components take exported fields, not functional options", f.Name.Name, ts.Name.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
