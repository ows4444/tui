package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// Example drives a whole build by hand and reads the summary from scrollback.
func Example() {
	s := tuitest.New(scripted{manualModel()}, 100, 40, tui.WithAltScreen(false))
	defer s.Close()
	for !s.Done() {
		s.Send(tickMsg{})
	}
	fmt.Println(strings.Contains(strings.Join(s.Screen(), "\n"), "Build succeeded"))
	// Output: true
}

func manualModel() model {
	m := newModel(3, "")
	m.manual = true
	return m
}
