package main

import (
	"fmt"
	"strings"
)

// Example builds the themefollow example's model and renders its first frame.
func Example() {
	fmt.Println(strings.TrimSpace(newModel().View()) != "")
	// Output: true
}
