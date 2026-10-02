package commandpalette

import (
	"github.com/ows4444/tui/ansi"
	"strconv"
)

// Linearize renders the palette as plain text for accessible output (see
// tui.Linearizer): the input's own line, then, while the dropdown is open,
// "N commands" and one line per command with its description, position and
// ", selected" on the highlighted one.
func (m Model) Linearize() string {
	out := m.Input.Linearize()
	filtered := m.filtered()
	if len(filtered) == 0 {
		return out
	}
	n := strconv.Itoa(len(filtered))
	if len(filtered) == 1 {
		out += "\n1 command"
	} else {
		out += "\n" + n + " commands"
	}
	for i, c := range filtered {
		line := ansi.Clean(m.Raw, c.Name)
		if c.Description != "" {
			line += ", " + ansi.Clean(m.Raw, c.Description)
		}
		line += ", option " + strconv.Itoa(i+1) + " of " + n
		if i == m.highlight {
			line += ", selected"
		}
		out += "\n" + line
	}
	return out
}
