package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example switches to the grid and reads the size its first cell was given.
func Example() {
	s := tuitest.New(newModel(), 120, 36)
	defer s.Close()
	s.Keys("2")
	s.WaitForText("1 39x3", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "1 39x3"))
	// Output: true
}
