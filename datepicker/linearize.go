package datepicker

// Linearize renders the calendar as plain text for accessible output (see
// tui.Linearizer): the date the cursor is on (unavailable if it is outside
// MinDate and MaxDate), the confirmed value if there is one, and the allowed
// range if one is set, in the words of Labels. The month grid is not spoken.
func (m Model) Linearize() string {
	l := m.labels()
	out := l.Picker + " " + m.speak(m.cursor)
	if !m.inRange(m.cursor) {
		out += ", " + l.Unavailable
	}
	if !m.Value.IsZero() {
		out += "\n" + l.Selected + ": " + m.speak(m.Value)
	}
	if !m.MinDate.IsZero() {
		out += "\n" + l.Earliest + ": " + m.speak(m.MinDate)
	}
	if !m.MaxDate.IsZero() {
		out += "\n" + l.Latest + ": " + m.speak(m.MaxDate)
	}
	return out
}
