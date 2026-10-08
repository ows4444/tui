package main

import (
	"fmt"
	"strings"
	"time"
)

// Example builds the pickers example's model and renders its first frame.
func Example() {
	m := newModel(time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC), ".")
	fmt.Println(strings.TrimSpace(m.View()) != "")
	// Output: true
}
