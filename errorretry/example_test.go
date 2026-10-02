package errorretry_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/errorretry"
)

// Enter asks for a retry until the allowed number is used up.
func Example() {
	m := errorretry.New("request failed", 1)
	m, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Printf("%T %d %v\n", cmd(), m.RetryCount(), m.Exhausted())
	// Output:
	// errorretry.RetryMsg 1 true
}
