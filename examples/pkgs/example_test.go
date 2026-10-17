package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example installs logr, which needs colorize (installed already).
func Example() {
	s := tuitest.New(newModel(true), 80, 24)
	defer s.Close()
	s.Keys("/", "logr", "enter", "i")
	s.WaitForText("Installed logr 3.0.1", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "installed"))
	// Output: true
}
