package viewport_test

import (
	"fmt"
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"strings"

	"github.com/ows4444/tui/viewport"
)

// A viewport shows a window onto longer content and scrolls it.
func Example() {
	m := viewport.New(10, 3)
	m.SetContent(strings.Join([]string{"one", "two", "three", "four", "five"}, "\n"))
	fmt.Println(m.View())
	m.LineDown(2)
	fmt.Println("--")
	fmt.Println(m.View())
	// Output:
	// one
	// two
	// three
	// --
	// three
	// four
	// five
}

// With Mouse on and Bounds set, a drag selects text and SelectedText (or the
// Copy key, "y") takes it.
func ExampleModel_SelectedText() {
	m := viewport.New(20, 3)
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 3}
	m.SetContent("alpha beta\ngamma delta")
	press := tui.MouseEvent{Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, X: 6, Y: 0}
	m, _ = m.Update(press)
	m, _ = m.Update(tui.MouseEvent{Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion, X: 4, Y: 1})
	m, _ = m.Update(tui.MouseEvent{Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease, X: 4, Y: 1})
	fmt.Println(m.SelectedText())
	// Output:
	// beta
	// gamma
}
