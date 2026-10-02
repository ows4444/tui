package autocomplete

import "strconv"

// Linearize renders the field and its open suggestions as plain text for
// accessible output (see tui.Linearizer): the input's own line, then, while
// the dropdown is open, "N suggestions" and one line per suggestion with its
// position and ", selected" on the highlighted one. With the dropdown closed
// it is just the input line.
func (m Model) Linearize() string {
	out := m.Input.Linearize()
	filtered := m.filtered()
	if len(filtered) == 0 {
		return out
	}
	n := strconv.Itoa(len(filtered))
	if len(filtered) == 1 {
		out += "\n1 suggestion"
	} else {
		out += "\n" + n + " suggestions"
	}
	for i, s := range filtered {
		line := s + ", option " + strconv.Itoa(i+1) + " of " + n
		if i == m.highlight {
			line += ", selected"
		}
		out += "\n" + line
	}
	return out
}
