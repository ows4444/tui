package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example follows the first link of the front page to the quick start.
func Example() {
	s := tuitest.New(newModel(sampleDocs()), 100, 28)
	defer s.Close()
	s.Keys("tab", "enter")
	s.WaitForText("Send a file", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Quick start"))
	// Output: true
}
