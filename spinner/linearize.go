package spinner

// Linearize renders the spinner as plain text for accessible output (see
// tui.Linearizer): its label and whether it is working, e.g. "Loading
// data, in progress" or "Spinner, stopped". The animated frame is not
// spoken, so the text is stable while it spins.
func (m Model) Linearize() string {
	label := m.Label
	if m.running {
		if label == "" {
			label = "Working"
		}
		return label + ", in progress"
	}
	if label == "" {
		label = "Spinner"
	}
	return label + ", stopped"
}
