package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example switches to the typewriter scene and reads its tab.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("3")
	s.WaitForText("▸3 Typewriter", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "▸3 Typewriter"))
	// Output: true
}
