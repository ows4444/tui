package form

import (
	"strconv"
	"strings"
)

// Linearize renders the form as plain text for accessible output (see
// tui.Linearizer): for a grouped form a line naming the group showing ("Group 2
// of 3, Billing") and its description, then one line per field of that group,
// in order. Each line is that field's own
// description (its label, kind, ", focused" when it has focus, and its value, a
// select's option and position, or a checkbox's checked state)
// followed by ", error: <message>" when the field has an error showing. A
// Secret field is a passwordinput, so it speaks only how many characters were
// entered, never the value. The error text is spoken as written, so a
// validator must not put a secret into its message.
func (m Model) Linearize() string {
	var lines []string
	if title, desc, pos, total := m.header(); len(m.groups) > 0 {
		head := ""
		if total > 1 {
			head = "Group " + strconv.Itoa(pos) + " of " + strconv.Itoa(total)
		}
		if title != "" {
			if head != "" {
				head += ", "
			}
			head += title
		}
		if head != "" {
			lines = append(lines, head)
		}
		if desc != "" {
			lines = append(lines, desc)
		}
	}
	lo, hi := m.span()
	for i := lo; i < hi; i++ {
		line := m.inputs[i].linearize()
		if d := m.inputs[i].desc; d != "" {
			line += ", " + d
		}
		if m.errs[i] != "" {
			line += ", error: " + m.errs[i]
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
