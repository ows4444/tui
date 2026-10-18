package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example hides the focused window with F5 and reads the task bar.
func Example() {
	m := newModel()
	m.manual = true
	s := tuitest.New(m, 100, 30)
	defer s.Close()
	s.Keys("f5")
	s.WaitForText("(hidden)", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Notes (hidden)"))
	// Output: true
}
