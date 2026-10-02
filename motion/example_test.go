package motion_test

import (
	"fmt"
	"time"

	"github.com/ows4444/tui/motion"
)

// A Tween turns elapsed time into a value. Drive it from a tui.Tick: keep the
// start time in the model and call At(time.Since(start)) on each tick. Under
// reduced motion it reports the end value straight away.
func ExampleTween() {
	slide := motion.Tween{From: 0, To: 20, Duration: 200 * time.Millisecond, Ease: motion.OutQuad}
	for _, ms := range []int{0, 50, 100, 200} {
		fmt.Printf("%3dms: %5.2f\n", ms, slide.At(time.Duration(ms)*time.Millisecond))
	}

	slide.Motion = motion.Reduced
	fmt.Printf("reduced at 0ms: %.0f\n", slide.At(0))
	// Output:
	// 0ms:  0.00
	//  50ms:  8.75
	// 100ms: 15.00
	// 200ms: 20.00
	// reduced at 0ms: 20
}
