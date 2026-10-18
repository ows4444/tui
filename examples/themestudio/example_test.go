package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example changes the base theme with "]" and reads the new name.
func Example() {
	s := tuitest.New(newModel(), 110, 32)
	defer s.Close()
	s.Keys("]")
	s.WaitForText("tokyo-night", 2e9)
	fmt.Println(!strings.Contains(strings.Join(s.Screen(), "\n"), "◂ tokyo-night"))
	// Output: true
}
