package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example types a name and clicks Greet, as a user with a mouse would.
func Example() {
	s := tuitest.New(initialModel(), 60, 16)
	defer s.Close()
	s.Keys("A", "d", "a")
	s.ClickText("Greet")
	s.WaitForText("Hello, Ada!", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Hello, Ada!"))
	// Output: true
}
