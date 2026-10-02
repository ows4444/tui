package main

import (
	"fmt"
	"github.com/ows4444/tui/ansi"
	"strings"
)

// Example builds the probe example's model and renders its first frame.
func Example() {
	m := newModel(ansi.TrueColor)
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
