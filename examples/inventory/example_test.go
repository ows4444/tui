package main

import (
	"fmt"
	"strings"
)

// Example builds the screen and renders its first frame.
func Example() {
	m := initialModel()
	fmt.Println(strings.TrimSpace(m.View()) != "", stock{}.Rows())
	// Output: true 5000
}
