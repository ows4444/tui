package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example opens Customers from the home screen and reads the breadcrumb.
func Example() {
	s := tuitest.New(newModel(), 100, 30)
	defer s.Close()
	s.Keys("enter")
	s.WaitForText("Home › Customers", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Home › Customers"))
	// Output: true
}
