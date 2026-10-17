package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example hides everything but errors.
func Example() {
	s := tuitest.New(sampleModel(), 120, 30)
	defer s.Close()
	s.Keys("1", "2", "3", "4")
	s.WaitForText("☐4 WARN", 2e9)
	out := strings.Join(s.Screen(), "\n")
	fmt.Println(strings.Contains(out, "ERROR"), strings.Contains(out, " INFO  "))
	// Output: true false
}
