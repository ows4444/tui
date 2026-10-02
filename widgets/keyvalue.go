package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// KV is a single label/value pair rendered by KeyValue.
type KV struct {
	Key   string
	Value string
}

// KeyValue renders pairs one per line as "key: value", padding keys with
// spaces (measured via ansi.Width) so every value starts at the same
// column regardless of key width. The key is styled distinctly from the
// value using t.Muted. An empty pairs slice renders as "".
func KeyValue(pairs []KV, t theme.Theme) string {
	if len(pairs) == 0 {
		return ""
	}

	width := 0
	for _, p := range pairs {
		if w := ansi.Width(p.Key); w > width {
			width = w
		}
	}

	keyStyle := ansi.NewStyle().Foreground(t.Muted).Bold()

	lines := make([]string, len(pairs))
	for i, p := range pairs {
		pad := strings.Repeat(" ", width-ansi.Width(p.Key)+1)
		lines[i] = keyStyle.Render(p.Key+":") + pad + p.Value
	}

	return strings.Join(lines, "\n")
}
