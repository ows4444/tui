package faces_test

import (
	"fmt"

	"github.com/ows4444/tui/faces"
)

// The catalog holds named faces; each has an animation and a pace.
func Example() {
	f := faces.Get(0)
	same, ok := faces.ByName(f.Name)
	fmt.Println(f.Name, f.Anim, ok, same.Name == f.Name)
	// Output:
	// Pip blink true true
}
