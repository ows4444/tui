package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const fixtureMod = "module example.com/m\n\ngo 1.25\n"

func TestListingIncludesOnlyExportedAPIWithoutComments(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod": fixtureMod,
		"a/a.go": `package a

// Doc comments must not appear.
type T struct {
	Public  int // trailing comment
	private int
}

func (T) Method() {}
func (T) hidden() {}

type impl struct{}

func (impl) Visible() {}
func (*impl) PtrVisible() {}

type G[K any] struct{}

func (G[K]) Generic() {}

func Func(x int) string { return "body-marker" }
func helper() {}

const Answer = 42
var hiddenVar = 1
`,
		"a/a_test.go":       "package a\n\nfunc TestOnly() {}\nfunc ExportedInTest() {}\n",
		"b/main.go":         "package main\n\nfunc Exported() {}\nfunc main() {}\n",
		"internal/i/i.go":   "package i\n\nfunc Secret() {}\n",
		"examples/e/e.go":   "package e\n\nfunc Demo() {}\n",
		"tools/t/t.go":      "package t\n\nfunc Tool() {}\n",
		"a/testdata/x/x.go": "package x\n\nfunc Fixture() {}\n",
	})
	b, err := listing(root)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"## example.com/m/a (package a)",
		"func Func(x int) string",
		"func (T) Method()",
		"func (G[K]) Generic()",
		"Public int",
		"const Answer = 42",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("listing missing %q:\n%s", want, got)
		}
	}
	for _, bad := range []string{
		"body-marker", "Visible", "PtrVisible", "comments", "private", "hidden", "helper", "TestOnly", "ExportedInTest",
		"Exported()", "Secret", "Demo", "Tool", "Fixture",
	} {
		if strings.Contains(got, bad) {
			t.Errorf("listing must not contain %q:\n%s", bad, got)
		}
	}
}

func TestListingIsDeterministicAndMergesBuildVariants(t *testing.T) {
	root := writeTree(t, map[string]string{
		"go.mod":      fixtureMod,
		"p/a_unix.go": "//go:build unix\n\npackage p\n\nfunc Same() {}\nfunc UnixOnly() {}\n",
		"p/a_win.go":  "//go:build windows\n\npackage p\n\nfunc Same() {}\nfunc WinOnly() {}\n",
	})
	first, err := listing(root)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := listing(root)
	if string(first) != string(second) {
		t.Error("two runs differ")
	}
	s := string(first)
	if strings.Count(s, "func Same()") != 1 || !strings.Contains(s, "UnixOnly") || !strings.Contains(s, "WinOnly") {
		t.Errorf("variants not merged once:\n%s", s)
	}
}

func TestFirstDifference(t *testing.T) {
	if got := firstDifference("a\nb\n", "a\nc\n"); !strings.Contains(got, "line 2") {
		t.Errorf("firstDifference = %q", got)
	}
	if got := firstDifference("same", "same"); got != "" {
		t.Errorf("identical inputs reported %q", got)
	}
}

func TestModulePathRequiresAModuleLine(t *testing.T) {
	root := writeTree(t, map[string]string{"go.mod": "go 1.25\n"})
	if _, err := modulePath(filepath.Join(root, "go.mod")); err == nil {
		t.Error("want an error for a go.mod without a module line")
	}
}
