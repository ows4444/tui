// Package boxdraw draws a bordered, padded layout.Box around pre-rendered
// text and returns the string, for the components that build their frame from
// a string instead of a layout.Node tree. It is the one place those components
// go through layout.BoxNode, so none of them calls the deprecated
// layout.Box.Render.
package boxdraw

import "github.com/ows4444/tui/layout"

// Draw returns content inside a box with the given border and pad cells of
// padding on every side. b carries the rest of the box (border colour,
// background); its own border, padding and width are replaced. A width above 0
// fixes the content width: narrower lines are padded to it and wider ones
// clipped. A width of 0 or less sizes the box to its widest line.
//
// Empty content still gets one blank row, as layout.Box.Render gives it: the
// minimum height below is the frame plus that row.
func Draw(b layout.Box, border layout.Border, pad, width int, content string) string {
	frame := 2 * pad
	if border != (layout.Border{}) {
		frame += 2
	}
	c := layout.Constraints{MaxW: layout.Unbounded, MinH: frame + 1, MaxH: layout.Unbounded}
	if width > 0 {
		c.MinW, c.MaxW = width+frame, width+frame
	}
	return layout.Draw(layout.BoxNode(b.Border(border).PaddingAll(pad), layout.Block(content)), c)
}
