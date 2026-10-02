package toast_test

import (
	"fmt"

	"github.com/ows4444/tui/toast"
)

// Show opens the toast and returns the Cmd that dismisses it after Duration.
func Example() {
	t := toast.New("Copied")
	fmt.Println(t.Open())
	cmd := t.Show()
	fmt.Println(t.Open(), cmd != nil)
	// Output:
	// false
	// true true
}
