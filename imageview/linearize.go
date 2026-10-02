package imageview

import "github.com/ows4444/tui/ansi"

// Linearize renders the image as plain text for accessible output (see
// tui.Linearizer): its alt text, or "Image" when there is none.
func (m Model) Linearize() string {
	if m.Width <= 0 || m.Height <= 0 {
		return ""
	}
	if a := asciiOnly(ansi.Sanitize(m.Alt)); a != "" {
		return a
	}
	return "Image"
}
