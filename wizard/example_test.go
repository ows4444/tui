package wizard_test

import (
	"errors"
	"fmt"

	"github.com/ows4444/tui/wizard"
)

// Next moves to the following step only when the step's validation passes.
func Example() {
	m := wizard.New("account", "profile", "done")
	err := m.Next(func() error { return errors.New("name is required") })
	fmt.Println(m.Current(), err)
	err = m.Next(func() error { return nil })
	fmt.Println(m.Current(), err)
	m.Back()
	fmt.Println(m.Current())
	// Output:
	// 0 name is required
	// 1 <nil>
	// 0
}
