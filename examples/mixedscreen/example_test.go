package main

import (
	"fmt"
	"strings"
)

// Example builds the mixedscreen example's model and renders its first frame.
func Example() {
	m := newScreen()
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
