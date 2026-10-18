// Command fireandfx is an animated splash made of three scenes: doom-fire, a
// plasma and a typewriter. Changing scene runs a transition from the motion/fx
// package (dissolve, fade, slide or sweep) over the whole region.
//
// Mouse: click a scene name to switch, click the effect name to change the
// transition. In the fire, press or drag to light it; in the plasma, move the
// pointer to move the ripple; in the typewriter, click to finish the text, and
// click again to type it over. Keyboard: 1 2 3 or Tab pick the scene, Space
// pauses, + and - change the speed, e changes the transition, i turns the
// fire's source on and off, Left and Right blow the fire, r restarts the
// scene, n steps by hand, ? shows help, q or Ctrl+C quits.
//
// With reduced motion on (REDUCE_MOTION=1 or NO_ANIMATION=1) nothing moves: the
// scenes show a finished frame, the transitions are instant and n steps by
// hand.
//
//	go run ./examples/fireandfx
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
