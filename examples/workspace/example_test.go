package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example moves focus to the panel and opens its second tab.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("tab", ".")
	s.WaitForText("▸Problems", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "3 problems"))
	// Output: true
}
