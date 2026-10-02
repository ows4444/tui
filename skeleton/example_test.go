package skeleton_test

import (
	"fmt"

	"github.com/ows4444/tui/skeleton"
)

// A skeleton is a placeholder shown while content loads; Start animates it.
func Example() {
	m := skeleton.New()
	fmt.Println(m.Running())
	_ = m.Start()
	fmt.Println(m.Running())
	// Output:
	// false
	// true
}
