package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixture makes a small project in a temporary folder, with fixed times and
// modes so a screen of it is the same on every machine.
func fixture(t testing.TB) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "project")
	files := map[string]string{
		"main.go": "package main\n", "util.go": "package main\n", "README.md": "# project\n", "notes.txt": "todo\n", "logo.png": "png",
		".env": "KEY=1\n", ".gitignore": "bin\n", "docs/guide.md": "# guide\n", "docs/.draft": "x", "src/lib.go": "package src\n", "data.bin": "0123456789",
	}
	when := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)
	for name, body := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = os.Chmod(p, 0o644)
		_ = os.Chtimes(p, when, when)
	}
	for _, d := range []string{"", "docs", "src"} {
		p := filepath.Join(root, d)
		_ = os.Chmod(p, 0o755)
		_ = os.Chtimes(p, when, when)
	}
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(filepath.Join(root, "empty"), when, when)
	return root
}
