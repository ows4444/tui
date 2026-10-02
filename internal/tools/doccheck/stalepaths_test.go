package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStalePathsNamesFileAndPath(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("ansi/ansi.go", "package ansi\n")
	write("README.md", "Run `go test ./ansi ./gone ./...`.\nSee github.com/ows4444/tui/ansi and github.com/ows4444/tui/removed.\n")
	write(".github/workflows/bench.yml", "run: go test ./ansi ./log\n")
	write("CHANGELOG.md", "removed ./log\n")

	got, err := stalePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, ".github/workflows/bench.yml") + ":1: ./log",
		filepath.Join(root, "README.md") + ":1: ./gone",
		filepath.Join(root, "README.md") + ":2: github.com/ows4444/tui/removed",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("stalePaths =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
