// Command snake is the classic: steer the snake to the apples, grow, and do not
// hit a wall or yourself. Five speed levels; an apple is worth the level.
//
// Mouse: click in the arena to turn toward the pointer, click the buttons in
// the top bar to pause, restart or change the speed. Keyboard: arrows, w a s d
// or h j k l steer, p or Space pauses, r restarts, + and - change the speed, ?
// shows help, q or Ctrl+C quits. The first move starts the game.
//
//	go run ./examples/snake
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
)

func main() {
	if _, err := tui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
