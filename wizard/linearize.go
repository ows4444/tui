package wizard

import (
	"strconv"
	"strings"
)

// Linearize renders the steps as plain text for accessible output (see
// tui.Linearizer): one line per step, "<title>, step K of N, completed",
// "current" or "upcoming". No connectors or markers.
func (m Model) Linearize() string {
	if len(m.titles) == 0 {
		return "No steps"
	}
	n := strconv.Itoa(len(m.titles))
	lines := make([]string, len(m.titles))
	for i, t := range m.titles {
		state := "upcoming"
		switch {
		case i < m.current:
			state = "completed"
		case i == m.current:
			state = "current"
		}
		lines[i] = t + ", step " + strconv.Itoa(i+1) + " of " + n + ", " + state
	}
	return strings.Join(lines, "\n")
}
