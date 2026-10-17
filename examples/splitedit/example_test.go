package main

import (
	"fmt"
	"strings"
)

// Example builds the screen and renders its first frame.
func Example() {
	m := initialModel()
	fmt.Println(strings.TrimSpace(m.View()) != "", strings.Count(m.edit.Value(), "return"))
	// Output: true 2
}
