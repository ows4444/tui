package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example sends a message in #general and waits for mira's reply.
func Example() {
	s := tuitest.New(newModel(true), 80, 24)
	defer s.Close()
	s.Keys("hello", "enter")
	s.WaitForText("pass that on", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "hello"))
	// Output: true
}
