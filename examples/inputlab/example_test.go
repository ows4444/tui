package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example types a key and reads how the lab describes it.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("ctrl+a")
	s.WaitForText("ctrl+a", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "ctrl+a"))
	// Output: true
}
