package main

import (
	"strings"
	"testing"
)

func TestPackageCommentsNamesBarePackages(t *testing.T) {
	root := t.TempDir()
	write(t, root, "good/a.go", "// Package good is documented.\npackage good\n")
	write(t, root, "intern/internal/inner/a.go", "package inner\n")
	write(t, root, "testonly/a_test.go", "// Package testonly is documented in a test file.\npackage testonly\n")
	write(t, root, "bare/a.go", "package bare\n")
	write(t, root, "bare/b.go", "package bare\n")
	write(t, root, "examples/demo/main.go", "package main\n")
	write(t, root, "testdata/x/a.go", "package x\n")

	got, err := packageComments(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"bare", "inner"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing finding for %q in:\n%s", want, joined)
		}
	}
	for _, bad := range []string{"good", "testonly", "demo", "testdata"} {
		if strings.Contains(joined, "/"+bad) {
			t.Errorf("unexpected finding for %q in:\n%s", bad, joined)
		}
	}
	if len(got) != 2 {
		t.Errorf("got %d findings, want 2:\n%s", len(got), joined)
	}
}
