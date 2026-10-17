package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/tuitest"
)

// Example counts the paid orders.
func Example() {
	s := tuitest.New(freshModel(), 110, 28)
	defer s.Close()
	s.Keys("ctrl+l", "SELECT COUNT(*) AS n FROM orders WHERE status = 'paid'", "enter")
	s.WaitForText("1 rows × 1 columns", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), " n"))
	// Output: true
}
