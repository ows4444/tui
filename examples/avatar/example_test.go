package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Example builds the gallery's model and reads the status line of its first
// frame: the selected name, its silhouette and its colours.
func Example() {
	lines := strings.Split(ansi.StripANSI(initialModel().View()), "\n")
	fmt.Println(lines[len(lines)-2])
	// Output: ada: nub, body #ad96f4, eyes #100e17  [8x4, none, blocks, none, own]
}
