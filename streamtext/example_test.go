package streamtext_test

import (
	"fmt"

	"github.com/ows4444/tui/streamtext"
)

// A streamtext reveals text progressively, like a model's streaming reply.
// Skip shows everything at once.
func Example() {
	m := streamtext.New()
	_ = m.SetText("hello, world")
	fmt.Println(m.Done())
	m.Skip()
	fmt.Println(m.Done(), m.Text())
	// Output:
	// false
	// true hello, world
}
