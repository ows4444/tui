package termpane_test

import (
	"fmt"

	"github.com/ows4444/tui/termpane"
)

// A pane for a shell, sized for a 100x30 area. The application calls SetSize
// when the layout changes, passes every Msg to Update and draws LayoutNode.
func Example() {
	pane := termpane.New("sh", "-l")
	pane.SetSize(100, 30)
	pane.Focus()
	cols, rows := pane.Size()
	fmt.Println(cols, rows, pane.Focused(), pane.Running())
	// Output: 100 30 true false
}
