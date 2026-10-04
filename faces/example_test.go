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
	// Any string picks a face of its own, the same one every time.
	fmt.Println(faces.For("ada").Name, faces.For(" ADA ").Name, faces.IndexFor("ada"))
	// Output:
	// Pip blink true true
	// Chip Chip 28
}
