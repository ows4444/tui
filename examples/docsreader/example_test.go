package main

import (
	"fmt"
	"strings"
)

// Example builds the reader and renders its first frame.
func Example() {
	m := initialModel()
	fmt.Println(strings.TrimSpace(m.View()) != "", len(m.reader.Headings()), len(m.reader.Links()))
	// Output: true 4 4
}
