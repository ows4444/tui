// Command stopwatch has three tabs: a stopwatch with laps, a countdown timer and
// a clock that shows the time in a few places. The stopwatch counts to a
// hundredth of a second and marks the fastest and slowest lap; the timer and the
// clocks are package clockview.
//
// Mouse: click a tab, click Start, Stop, Lap or Reset, click a timer length to
// pick it, click - or + to change it by a minute, scroll the laps. Keyboard: 1 2
// 3 or Tab change tab, Space starts and stops, l takes a lap, r resets, on the
// timer Up and Down change the length by a minute and p steps through 1, 5, 10
// and 25 minutes, ? shows help, q or Ctrl+C quits.
//
//	go run ./examples/stopwatch
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	m := newModel()
	if _, err := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
