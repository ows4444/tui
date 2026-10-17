package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example picks a two-night stay and books it.
func Example() {
	s := tuitest.New(initialModel(now), 80, 24)
	defer s.Close()
	s.Keys("enter", "right", "right", "enter", "tab", "tab", "tab", "enter")
	s.WaitForText("Booked", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Booked: 14 Oct to 16 Oct"))
	// Output: true
}
