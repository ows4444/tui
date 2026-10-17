package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example signs up on the free plan and reads the review.
func Example() {
	s := tuitest.New(initialModel(), 80, 24)
	defer s.Close()
	s.Keys("Ada", "tab", "ada@example.com", "tab", "longenough", "enter")
	s.Keys("tab", "tab", "space", "enter") // plan stays Free, accept the terms
	s.WaitForText("Create account", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Ada"))
	// Output: true
}
