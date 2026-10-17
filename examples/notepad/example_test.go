package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example types into an empty document and saves it as hello.txt.
func Example() {
	st := memStore{}
	s := tuitest.New(newModel(st, ""), 80, 24)
	defer s.Close()
	s.Keys("hi", "ctrl+s")
	s.Keys("hello.txt", "enter")
	s.WaitForText("Saved hello.txt", 2e9)
	fmt.Println(strings.TrimSpace(st["hello.txt"]))
	// Output: hi
}
