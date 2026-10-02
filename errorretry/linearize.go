package errorretry

import "strconv"

// Linearize renders the error as plain text for accessible output (see
// tui.Linearizer): the message, how many retries have been used, and the
// keys that are available, e.g. "Error: connection lost", "Retries used: 1
// of 3", "Press Enter or r to retry, Escape to dismiss".
func (m Model) Linearize() string {
	used := "Retries used: " + strconv.Itoa(m.retryCount) + " of " + strconv.Itoa(m.MaxRetries)
	action := "Press Enter or r to retry, Escape to dismiss"
	if m.Exhausted() {
		action = "No retries left. Press Escape to dismiss"
	}
	return "Error: " + m.Message + "\n" + used + "\n" + action
}
