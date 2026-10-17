package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example types a hex code and reads the contrast against white.
func Example() {
	s := tuitest.New(initialModel(), 80, 24)
	defer s.Close()
	s.Keys("tab", "tab", "tab", "ctrl+u", "#000000")
	s.WaitForText("1.00", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "fail"))
	// Output: true
}
