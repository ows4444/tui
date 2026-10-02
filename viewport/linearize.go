package viewport

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Linearize renders the whole content as plain text for accessible output
// (see tui.Linearizer): every line, escape sequences removed, with none of
// the windowing. Scrolling has no meaning in a linear transcript, and the
// content is what a reader wants. There is no header, so appending content
// only adds new lines to the transcript.
func (m Model) Linearize() string {
	out := make([]string, len(m.lines))
	for i, l := range m.lines {
		out[i] = ansi.StripANSI(l)
	}
	return strings.Join(out, "\n")
}
