package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// Example runs the lobby as one session of a hub and reads who is online.
func Example() {
	l, _ := newLobby(newHub(5), "ada", "10.0.0.1:1", func(tui.Msg) bool { return true })
	s := tuitest.New(l, 80, 24)
	defer s.Close()
	s.WaitForText("Online (1)", 2e9)
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "ada (you)"))
	// Output: true
}
