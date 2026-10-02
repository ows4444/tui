package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// blockWidth is the width of a (possibly multi-line) block's widest line.
func blockWidth(block string) int {
	width := 0
	for _, l := range strings.Split(block, "\n") {
		if w := ansi.Width(l); w > width {
			width = w
		}
	}
	return width
}
