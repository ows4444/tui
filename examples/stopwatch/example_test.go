package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example opens the clock tab and reads the zones.
func Example() {
	s := tuitest.New(newModel(), 80, 24)
	defer s.Close()
	s.Keys("3")
	s.WaitForText("Tokyo", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "New York"))
	// Output: true
}
