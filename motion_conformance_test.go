package tui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// tickAllowlist names the library packages that schedule tick messages but
// have no reduced-motion hook, each with the reason none is needed. A package
// that schedules ticks must either read a motion.Preference (call Reduced) or
// be listed here.
var tickAllowlist = map[string]string{
	"clipboard":    "the tick is a dismiss timeout for a status line, not an animation",
	"clockview":    "the tick refreshes the time shown, which is content, not decoration",
	"toast":        "the tick is the auto-dismiss timeout; the toast itself does not animate",
	"toolapproval": "the tick is a decision deadline, not an animation",
}

// scanTickPackages walks the library packages under root and returns, for
// every package that schedules a tick (tui.Tick or motion.After), whether one of its non-test files also
// calls a Reduced method (the motion.Preference hook).
func scanTickPackages(t *testing.T, root string) map[string]bool {
	t.Helper()
	hooked := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch name := d.Name(); {
			case path == root:
			case name == "examples" || name == "tools" || name == "internal" || name == "testdata" || name == "motion" || strings.HasPrefix(name, "."):
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
		ticks, reduced := false, false
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && ((x.Name == "tui" && sel.Sel.Name == "Tick") || (x.Name == "motion" && sel.Sel.Name == "After")) {
				ticks = true
			}
			if sel.Sel.Name == "Reduced" {
				reduced = true
			}
			return true
		})
		if ticks {
			hooked[pkg] = hooked[pkg] || reduced
		} else if reduced {
			if _, seen := hooked[pkg]; !seen {
				hooked[pkg] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hooked
}

// unhookedTickers lists the packages that schedule ticks with neither a
// reduced-motion hook nor an allowlist entry, sorted.
func unhookedTickers(t *testing.T, root string, allow map[string]string) []string {
	t.Helper()
	var bad []string
	for pkg, ok := range scanTickPackages(t, root) {
		if !ok && allow[pkg] == "" {
			bad = append(bad, pkg)
		}
	}
	sort.Strings(bad)
	return bad
}

func TestEveryTickSchedulerIsHookedOrAllowlisted(t *testing.T) {
	if bad := unhookedTickers(t, ".", tickAllowlist); len(bad) > 0 {
		t.Fatalf("these packages schedule ticks with no motion.Preference hook and no allowlist entry: %s", strings.Join(bad, ", "))
	}
}

// Every allowlist entry is still needed: a package that is hooked, or no longer
// ticks, must be taken off the list.
func TestTickAllowlistHasNoStaleEntries(t *testing.T) {
	seen := scanTickPackages(t, ".")
	for pkg, why := range tickAllowlist {
		hooked, ticks := seen[pkg]
		switch {
		case !ticks:
			t.Errorf("allowlisted package %q no longer schedules ticks", pkg)
		case hooked:
			t.Errorf("allowlisted package %q has a hook now; remove it from the list", pkg)
		case strings.TrimSpace(why) == "":
			t.Errorf("allowlisted package %q has no reason", pkg)
		}
	}
}

// The rule bites: on a scratch tree, a scheduler with no hook fails naming its
// package, adding the hook or an allowlist entry makes it pass, and removing
// the hook again makes it fail.
func TestConformanceFailsWhenAHookIsRemoved(t *testing.T) {
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
	const unhooked = "package fancy\n\nimport \"github.com/ows4444/tui\"\n\nfunc F() tui.Cmd { return tui.Tick(0, nil) }\n"
	const hooked = "package fancy\n\nimport \"github.com/ows4444/tui\"\n\ntype M struct{ Motion interface{ Reduced() bool } }\n\nfunc (m M) F() tui.Cmd {\n\tif m.Motion.Reduced() {\n\t\treturn nil\n\t}\n\treturn tui.Tick(0, nil)\n}\n"

	write(unhooked)
	if bad := unhookedTickers(t, root, nil); len(bad) != 1 || bad[0] != "fancy" {
		t.Fatalf("unhooked scheduler: got %v, want [fancy]", bad)
	}
	if bad := unhookedTickers(t, root, map[string]string{"fancy": "a reason"}); len(bad) != 0 {
		t.Errorf("allowlisted scheduler still reported: %v", bad)
	}
	write(hooked)
	if bad := unhookedTickers(t, root, nil); len(bad) != 0 {
		t.Errorf("hooked scheduler reported: %v", bad)
	}
	write(unhooked) // the hook removed again
	if bad := unhookedTickers(t, root, nil); len(bad) != 1 {
		t.Errorf("removing the hook did not make the check fail: %v", bad)
	}
}

// TestNoLibraryPackageCallsTuiTick proves criterion #36: library packages
// schedule timers through motion, not tui.Tick.
func TestNoLibraryPackageCallsTuiTick(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch name := d.Name(); {
			case path == ".":
			case name == "examples" || name == "motion" || name == "testdata" || strings.HasPrefix(name, "."):
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Dir(path) == "." || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "tui.Tick(") {
			t.Errorf("%s calls tui.Tick; use motion.After or a motion.Clock", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestNoExampleCallsTuiTick proves criterion #109: examples schedule timers
// through motion, so they show the pattern the library recommends.
func TestNoExampleCallsTuiTick(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir("examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Tick" {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "tui" {
					t.Errorf("%s: tui.Tick; use motion.After or a motion.Clock", fset.Position(sel.Pos()))
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

// TestExamplesBuild proves criterion #110: every example compiles.
func TestExamplesBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds every example")
	}
	if out, err := exec.Command("go", "build", "./examples/...").CombinedOutput(); err != nil {
		t.Fatalf("go build ./examples/...: %v\n%s", err, out)
	}
}
