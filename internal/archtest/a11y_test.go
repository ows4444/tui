package archtest

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

// a11yAllowlist names the stateful widget packages that legitimately lack
// Linearize or LayoutNode, with the reason. Key is the module-relative
// package path; the value maps each missing method to its reason. Empty today:
// every stateful widget package has both. Adding an entry needs a real reason
// (a widget with no text content, say), not "not done yet".
var a11yAllowlist = map[string]map[string]string{}

// a11yAlsoRequired lists packages whose Model has no Update method (so the
// stateful rule would miss them) but which must still have both methods.
var a11yAlsoRequired = map[string]bool{"markdown": true, "wizard": true, "imageview": true, "notificationcenter": true}

var a11yMethods = []string{"Linearize", "LayoutNode"}

type a11yPkg struct {
	hasModel, hasUpdate bool
	methods             map[string]bool
}

// scanA11y parses the non-test files of every library package under root
// (skipping the root package, examples, internal, tools, testdata and dot
// directories) and records the Model type, its Update method and its
// Linearize and LayoutNode methods.
func scanA11y(t *testing.T, root string) map[string]*a11yPkg {
	t.Helper()
	pkgs := map[string]*a11yPkg{}
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
		pkg = filepath.ToSlash(pkg)
		p := pkgs[pkg]
		if p == nil {
			p = &a11yPkg{methods: map[string]bool{}}
			pkgs[pkg] = p
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.Name == "Model" {
						p.hasModel = true
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil || len(d.Recv.List) == 0 || recvName(d.Recv.List[0].Type) != "Model" {
					continue
				}
				if d.Name.Name == "Update" {
					p.hasUpdate = true
				}
				p.methods[d.Name.Name] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkgs
}

func recvName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.StarExpr:
		return recvName(e.X)
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		return recvName(e.X)
	}
	return ""
}

// a11yGaps returns "pkg: Method" for each stateful package lacking a required
// method with no allowlist entry, sorted.
func a11yGaps(pkgs map[string]*a11yPkg, allow map[string]map[string]string, also map[string]bool) []string {
	var bad []string
	for name, p := range pkgs {
		if !p.hasModel || !(p.hasUpdate || also[name]) {
			continue
		}
		for _, m := range a11yMethods {
			if !p.methods[m] && allow[name][m] == "" {
				bad = append(bad, name+": "+m)
			}
		}
	}
	sort.Strings(bad)
	return bad
}

func TestEveryStatefulWidgetHasLinearizeAndLayoutNode(t *testing.T) {
	root := moduleRoot(t)
	if bad := a11yGaps(scanA11y(t, root), a11yAllowlist, a11yAlsoRequired); len(bad) > 0 {
		t.Fatalf("stateful widget packages missing a Model method (implement it, or add an a11yAllowlist entry with a reason):\n  %s", strings.Join(bad, "\n  "))
	}
}

// Every allowlist entry is still needed and carries a reason.
func TestA11yAllowlistHasNoStaleEntries(t *testing.T) {
	pkgs := scanA11y(t, moduleRoot(t))
	for name, ms := range a11yAllowlist {
		p := pkgs[name]
		if p == nil || !p.hasModel {
			t.Errorf("allowlisted package %q has no Model", name)
			continue
		}
		for m, why := range ms {
			switch {
			case p.methods[m]:
				t.Errorf("%s now has %s; remove it from the allowlist", name, m)
			case strings.TrimSpace(why) == "":
				t.Errorf("%s: %s allowlisted without a reason", name, m)
			}
		}
	}
	for name := range a11yAlsoRequired {
		if p := pkgs[name]; p == nil || !p.hasModel {
			t.Errorf("a11yAlsoRequired package %q has no Model", name)
		}
	}
}

// The rule bites: on a scratch tree, a stateful Model missing a method fails,
// an allowlist entry or the method makes it pass.
func TestA11yRuleDetectsGaps(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "fancy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "f.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const base = "package fancy\n\ntype Model struct{}\n\nfunc (m *Model) Update(x int) (Model, int) { return *m, 0 }\n"
	write(base)
	got := a11yGaps(scanA11y(t, root), nil, nil)
	if len(got) != 2 || got[0] != "fancy: LayoutNode" || got[1] != "fancy: Linearize" {
		t.Fatalf("gaps = %v", got)
	}
	allow := map[string]map[string]string{"fancy": {"Linearize": "no text", "LayoutNode": "fixed size"}}
	if got := a11yGaps(scanA11y(t, root), allow, nil); len(got) != 0 {
		t.Errorf("allowlisted gaps reported: %v", got)
	}
	write(base + "func (m Model) Linearize() string { return \"\" }\nfunc (m Model) LayoutNode() int { return 0 }\n")
	if got := a11yGaps(scanA11y(t, root), nil, nil); len(got) != 0 {
		t.Errorf("complete package reported: %v", got)
	}
	// A Model with no Update is not stateful unless listed.
	write("package fancy\n\ntype Model struct{}\n")
	if got := a11yGaps(scanA11y(t, root), nil, nil); len(got) != 0 {
		t.Errorf("non-stateful package reported: %v", got)
	}
	if got := a11yGaps(scanA11y(t, root), nil, map[string]bool{"fancy": true}); len(got) != 2 {
		t.Errorf("also-required package: %v", got)
	}
}

// Charts are stateless functions, so each gets a Linearize summary function in
// place of a Model method.
func TestChartsHaveLinearizeSummaries(t *testing.T) {
	root := moduleRoot(t)
	have := map[string]bool{}
	dir := filepath.Join(root, "widgets", "chart")
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil {
				have[fd.Name.Name] = true
			}
		}
	}
	for _, c := range []string{"Sparkline", "LineChart", "BarChart", "Gauge", "HeatMap"} {
		if !have[c] {
			t.Errorf("chart %s no longer exists; update this test", c)
		} else if !have["Linearize"+c] {
			t.Errorf("chart %s has no Linearize%s summary", c, c)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}
