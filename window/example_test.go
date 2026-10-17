package window_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/window"
)

// Two windows on a 40x12 desktop; dragging the title bar of the back one moves
// it and brings it to the front.
func Example() {
	m := window.New(
		window.Window{ID: "files", Title: "Files", Rect: hittest.Rect{X: 1, Y: 1, W: 16, H: 6}, Content: layout.Block("a.txt\nb.txt")},
		window.Window{ID: "log", Title: "Log", Rect: hittest.Rect{X: 10, Y: 3, W: 18, H: 6}},
	)
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 40, H: 12}
	m, _ = m.Update(tui.MouseEvent{X: 4, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	m, _ = m.Update(tui.MouseEvent{X: 14, Y: 5, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion})
	m, _ = m.Update(tui.MouseEvent{X: 14, Y: 5, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	w, _ := m.Window("files")
	fmt.Println(m.Focused(), w.Rect.X, w.Rect.Y)
	// Output: files 11 5
}
