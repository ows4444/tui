package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example flips Word wrap in the Editor category and applies it.
func Example() {
	s := tuitest.New(initialModel(), 80, 24)
	defer s.Close()
	s.Keys("pgdown", "pgdown", "down", "space", "ctrl+s")
	s.WaitForText("Applied 1 change", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "All changes applied"))
	// Output: true
}
