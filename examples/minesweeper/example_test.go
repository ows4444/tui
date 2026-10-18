package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example opens the first cell with Enter, which is always safe, and reads the status.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("enter")
	s.WaitForText("cells to open", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "cells to open"))
	// Output: true
}
