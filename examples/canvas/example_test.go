package main

import (
	"fmt"
	"strings"
)

// Example builds the canvas example's model and renders its first frame.
func Example() {
	m := canvas{}
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
