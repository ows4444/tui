package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example opens a folder with Enter and reads its path bar.
func Example() {
	dir, _ := os.MkdirTemp("", "dialog")
	defer os.RemoveAll(dir)
	_ = os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "docs", "a.txt"), []byte("x"), 0o644)
	s := tuitest.New(newModel(dir, false), 100, 30)
	defer s.Close()
	s.Keys("enter")
	s.WaitForText("a.txt", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "docs"))
	// Output: true
}
