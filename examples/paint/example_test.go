package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example picks the rectangle tool and reads it back in the tool row.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("r")
	s.WaitForText("▸Rect", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "▸Rect"))
	// Output: true
}
