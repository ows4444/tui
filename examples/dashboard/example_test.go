package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example zooms the network panel.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("tab", "tab", "tab", "enter")
	s.WaitForText("Network (zoomed)", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "since start"))
	// Output: true
}
