package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example signs in with the demo account.
func Example() {
	s := tuitest.New(newModel(true), 80, 24)
	defer s.Close()
	s.Keys("ada", "enter", "lovelace1815", "enter")
	s.WaitForText("Signed in", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Welcome back, ada."))
	// Output: true
}
