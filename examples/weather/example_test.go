package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example searches for Oslo and opens it.
func Example() {
	f := fastFixture()
	m := newModel(f, 0, "")
	m.now = f.now
	s := tuitest.New(m, 100, 28)
	defer s.Close()
	s.Keys("oslo")
	s.WaitForText("› Oslo, Norway", 2e9)
	s.Keys("enter")
	s.WaitForText("fetched 09:30:00", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Now "))
	// Output: true
}
