// Command desktop is a small windowing system in the terminal: four windows
// (a clock, a counter, a notepad and an event log) that you drag by the title
// bar, resize by an edge, stack, hide to a task bar, close and bring back, and a
// modal dialog that blocks the rest until you answer. It is built on package
// window, which owns the rectangles and the order; the program supplies what is
// inside each window and routes the clicks and keys that land there.
//
// Mouse: drag a title bar to move a window, drag its left, right or bottom edge
// to resize, press a window to raise it, click the close box (x) to close it,
// click a task bar entry to raise, hide or show a window, click + there to bring
// back a closed one, click the counter's buttons, scroll the log. Keyboard: Tab
// and Shift+Tab change window, F2 maximizes, F3 brings back a closed window, F4
// closes, F5 hides, F6 starts move mode (arrows move, Shift+arrows resize, Esc
// leaves), Alt with the arrows moves a window without it, F1 shows help, Ctrl+Q
// quits (it asks first). Type in the notepad when it has focus; because of that
// q is not a quit key here.
//
//	go run ./examples/desktop
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
