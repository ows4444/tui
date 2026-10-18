package main

import (
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/spinner"
)

// calm makes the spinner stand still when the user asked for less motion
// (REDUCE_MOTION or NO_ANIMATION); it then shows its last frame and ticks no more.
func calm(s spinner.Model) spinner.Model {
	s.Motion = motion.Detect()
	return s
}
