package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example opens a file and jumps to its last line.
func Example() {
	s := tuitest.New(newModel(project(), "demo"), 100, 24)
	defer s.Close()
	s.Keys("tab", "G")
	s.WaitForText("Ln ", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "main.go"))
	// Output: true
}
