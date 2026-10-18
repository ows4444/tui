package avatar

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Linearize renders the avatar as plain text for accessible output (see
// tui.Linearizer): "Avatar: " and the name, or "Avatar" when the name is
// blank, followed by the expression when one is set, as in "Avatar: ada,
// thinking". The picture itself is not described. A Decorative avatar, set
// when a label next to it already says the name, returns "".
func (m Model) Linearize() string {
	if m.Decorative || m.Width <= 0 || m.Height <= 0 {
		return ""
	}
	out := "Avatar"
	if n := strings.TrimSpace(ansi.Sanitize(m.Name)); n != "" {
		out += ": " + n
	}
	if m.Expression.pose().name != poses[ExpressionNone].name {
		out += ", " + m.Expression.String()
	}
	return out
}
