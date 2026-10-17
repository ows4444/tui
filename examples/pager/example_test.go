package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example searches the sample for a phrase.
func Example() {
	m := newModel("sample").begin(strings.NewReader(sample()), false)
	s := tuitest.New(m, 100, 24)
	defer s.Close()
	s.WaitForText("PAGER SAMPLE", 2e9)
	s.Keys("/", "Section 9", "enter")
	s.WaitForText("1 matches", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Section 9"))
	// Output: true
}
