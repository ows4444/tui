package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example edits the first city and saves the file.
func Example() {
	st := memStore{"c.csv": "name,n\nA,1\nB,2\n"}
	m, _ := newModel(st, "c.csv")
	s := tuitest.New(m, 80, 24)
	defer s.Close()
	s.Keys("enter", "ctrl+u", "Z", "enter", "ctrl+s")
	s.WaitForText("Saved c.csv", 2e9)
	fmt.Println(strings.Contains(st["c.csv"], "Z,1"))
	// Output: true
}
