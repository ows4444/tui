// Command minesweeper is the classic mine field in three sizes. Open every
// cell that is not a mine; a number says how many mines touch that cell. The
// first click is always safe and opens an area.
//
// Mouse: left click opens a cell, right click flags it, middle click (or a
// double-click) on a number opens its neighbours when the flags match,
// the buttons in the top bar start a new game or change the size. Keyboard:
// arrows or h j k l move, Space or Enter opens (on a number: chords), f flags,
// c chords, 1 2 3 pick the size, n starts a new game, ? shows help, q or
// Ctrl+C quits.
//
//	go run ./examples/minesweeper
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	if _, err := ui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
