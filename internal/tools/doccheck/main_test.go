package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, src string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckNamesUndocumented(t *testing.T) {
	root := t.TempDir()
	write(t, root, "pkg/a.go", `package pkg

// Documented is fine.
func Documented() {}

func Bare() {}

type T struct{}

func (T) Method() {}

// Grouped covers its members.
const (
	A = 1
	B = 2
)

var (
	// C is documented.
	C = 1
	D = 2
)

func unexported() {}

type hidden struct{}

func (hidden) Exported() {}
`)
	got, err := check(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, g := range got {
		if !strings.Contains(g, "a.go:") {
			t.Errorf("finding lacks file and line: %q", g)
		}
		names = append(names, g[strings.LastIndex(g, " ")+1:])
	}
	want := "Bare T T.Method D"
	if strings.Join(names, " ") != want {
		t.Errorf("got %v, want %s (in source order)", names, want)
	}
	if !strings.Contains(got[0], "a.go:6:") {
		t.Errorf("Bare should be reported at line 6: %q", got[0])
	}
}

func TestCheckSkipsNonPublic(t *testing.T) {
	root := t.TempDir()
	bare := "package x\n\nfunc Bare() {}\n"
	write(t, root, "examples/e/x.go", bare)
	write(t, root, "internal/i/x.go", bare)
	write(t, root, "tools/t/x.go", bare)
	write(t, root, "pkg/x_test.go", bare)
	write(t, root, "cmd/main.go", "package main\n\nfunc Bare() {}\n")
	got, err := check(root)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want nothing", got, err)
	}
}

func TestTreeIsDocumented(t *testing.T) {
	got, err := check(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("undocumented identifiers:\n%s", strings.Join(got, "\n"))
	}
}
