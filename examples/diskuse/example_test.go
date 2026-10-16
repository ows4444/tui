package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Example lists a directory with one file in it.
func Example() {
	dir, _ := os.MkdirTemp("", "diskuse")
	defer os.RemoveAll(dir)
	os.WriteFile(filepath.Join(dir, "notes.txt"), make([]byte, 2048), 0o644)
	m := newModel(dir)
	fmt.Println(strings.Contains(m.View(), "2.0 KB"), strings.Contains(m.View(), "notes.txt"))
	// Output: true true
}
