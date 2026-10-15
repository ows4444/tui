package main

import (
	"fmt"
	"strings"
)

// Example builds the explorer's model from a small document and shows that the
// first screen names the source.
func Example() {
	m, _ := newModel([]byte(`{"a":1}`), "a.json")
	fmt.Println(strings.Contains(m.View(), "a.json"))
	// Output: true
}
