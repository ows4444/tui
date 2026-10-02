package widgets

import (
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Card renders content the same way Panel does — a reverse-styled title
// row separated from the body by a divider — but always draws a rounded
// border, regardless of t.Border, for callers that want a Card's rounded
// look consistently even under a theme configured with square or double
// borders elsewhere. width behaves exactly as it does for Box and Panel.
func Card(title, content string, t theme.Theme, width int) string {
	border := layout.RoundedBorder()
	if t.Border == layout.ASCIIBorder() {
		border = layout.ASCIIBorder() // an ASCII theme has no rounded corners to draw
	}
	return panel(title, content, t, width, border)
}
