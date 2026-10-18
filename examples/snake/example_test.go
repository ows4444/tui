package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example starts the game with an arrow key and reads the paused banner.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("up", "p")
	s.WaitForText("Paused", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Paused"))
	// Output: true
}
