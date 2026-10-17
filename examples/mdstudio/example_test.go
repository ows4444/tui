package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example types a new heading and sees it in the outline.
func Example() {
	st := memStore{"n.md": "# One\n\ntext\n"}
	m, _ := newModel(st, "n.md")
	s := tuitest.New(m, 120, 30)
	defer s.Close()
	s.Keys("ctrl+end", "## Two")
	s.WaitForText("Two", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "  Two"))
	// Output: true
}
