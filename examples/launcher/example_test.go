package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example runs the "fmt" command of the sample project.
func Example() {
	s := tuitest.New(newModel(sampleCommands(), false, 0), 80, 24)
	defer s.Close()
	s.Keys("fmt", "enter")
	s.WaitForText("3 files changed", 2e9)
	s.WaitForText("exit 0", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "✓ exit 0"))
	// Output: true
}
