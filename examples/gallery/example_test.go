package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ows4444/tui/tuitest"
)

// Example opens the dialog from the Menus page and reads its message.
func Example() {
	s := tuitest.New(newModel(), 100, 28)
	defer s.Close()
	s.Keys("d")
	s.WaitForText("This cannot be undone.", 2*time.Second)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Delete file"))
	// Output: true
}
