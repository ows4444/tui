package form

import "strings"

// Linearize renders the form as plain text for accessible output (see
// tui.Linearizer): one line per field, in order. Each line is that field's own
// description (its label, kind, ", focused" when it has focus, and its value, a
// select's option and position, or a checkbox's checked state)
// followed by ", error: <message>" when the field has an error showing. A
// Secret field is a passwordinput, so it speaks only how many characters were
// entered, never the value. The error text is spoken as written, so a
// validator must not put a secret into its message.
func (m Model) Linearize() string {
	lines := make([]string, len(m.inputs))
	for i, in := range m.inputs {
		line := in.linearize()
		if m.errs[i] != "" {
			line += ", error: " + m.errs[i]
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}
