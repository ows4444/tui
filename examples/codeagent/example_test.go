package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example asks for an explanation, which needs no tool, and reads the reply.
func Example() {
	m := newModel()
	m.still = true
	s := tuitest.New(m, 100, 30)
	defer s.Close()
	s.Paste("explain the layout package")
	s.Keys("enter")
	s.WaitForText("Named nodes", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Named nodes"))
	// Output: true
}
