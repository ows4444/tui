package main

import (
	"fmt"
	"strings"
)

// Example builds the cursorfield example's model and renders its first frame.
func Example() {
	m := initialModel()
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
