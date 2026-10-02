package appshell

import "strings"

// Linearize renders the shell as plain text for accessible output (see
// tui.Linearizer): the title, the input's line, the content's lines, and each
// key hint as "Key: <key>, <action>".
func (m Model) Linearize() string {
	lines := []string{m.Title, m.Input.Linearize()}
	if c := m.Content.Linearize(); c != "" {
		lines = append(lines, c)
	}
	for _, h := range m.Hints {
		lines = append(lines, "Key: "+h.Key+", "+h.Action)
	}
	return strings.Join(lines, "\n")
}
