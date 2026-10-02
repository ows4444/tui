package datepicker

// dateLayout reads naturally aloud: "Monday, January 2, 2006".
const dateLayout = "Monday, January 2, 2006"

// Linearize renders the calendar as plain text for accessible output (see
// tui.Linearizer): the date the cursor is on (", unavailable" if it is
// outside MinDate and MaxDate), the confirmed value if there is one, and the
// allowed range if one is set. The month grid is not spoken.
func (m Model) Linearize() string {
	out := "Date picker, cursor on " + m.cursor.Format(dateLayout)
	if !m.inRange(m.cursor) {
		out += ", unavailable"
	}
	if !m.Value.IsZero() {
		out += "\nSelected date: " + m.Value.Format(dateLayout)
	}
	if !m.MinDate.IsZero() {
		out += "\nEarliest date: " + m.MinDate.Format(dateLayout)
	}
	if !m.MaxDate.IsZero() {
		out += "\nLatest date: " + m.MaxDate.Format(dateLayout)
	}
	return out
}
