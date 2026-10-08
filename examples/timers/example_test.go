package main

import (
	"fmt"
	"strings"
)

// Example builds the timers example's model and renders its first frame.
func Example() {
	m := newModel(nine)
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
