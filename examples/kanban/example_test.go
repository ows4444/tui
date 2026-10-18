package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example moves the selected card to the Doing column with ">" and reads the
// new count.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys(">")
	s.WaitForText("Doing (3)", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Doing (3)"))
	// Output: true
}
