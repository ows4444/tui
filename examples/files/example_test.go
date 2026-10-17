package main

import (
	"fmt"

	"github.com/ows4444/tui/tuitest"
)

// Example opens the docs folder by clicking it in the list and double-clicking.
func Example() {
	m := newModel(sample(), "sample")
	s := tuitest.New(m, 100, 30)
	defer s.Close()
	s.ClickText("docs/")
	s.Keys("enter")
	fmt.Println(s.WaitForText("guide.md", 2e9))
	// Output: true
}
