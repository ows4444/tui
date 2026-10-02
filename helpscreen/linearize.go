package helpscreen

import "strconv"

// Linearize renders the help as plain text for accessible output (see
// tui.Linearizer): "Help, N key bindings" and one "Key: <key>, <action>" line
// per binding, or "" while it is closed.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	out := "Help, " + strconv.Itoa(len(m.Hints)) + " key bindings"
	if len(m.Hints) == 1 {
		out = "Help, 1 key binding"
	}
	for _, h := range m.Hints {
		out += "\nKey: " + h.Key + ", " + h.Action
	}
	return out
}
