package main

import (
	"fmt"
	"strings"
)

// Example builds the tutorial's model and renders its first frame.
func Example() {
	fmt.Println(strings.Contains(newModel().View(), "Tasks"))
	// Output: true
}
