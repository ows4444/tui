package avatar

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Linearize renders the avatar as plain text for accessible output (see
// tui.Linearizer): "Avatar: " and the name, or "Avatar" when the name is
// blank. The picture is decoration and is not described.
func (m Model) Linearize() string {
	if m.Width <= 0 || m.Height <= 0 {
		return ""
	}
	if n := strings.TrimSpace(ansi.Sanitize(m.Name)); n != "" {
		return "Avatar: " + n
	}
	return "Avatar"
}
