package clockview

// Linearize renders the clock as plain text for accessible output (see
// tui.Linearizer): its kind and reading, and for a stopwatch or timer
// whether it is running, e.g. "Timer, 00:09 remaining, running". The
// reading is the same text View shows.
func (m Model) Linearize() string {
	state := "stopped"
	if m.running {
		state = "running"
	}
	switch m.Mode {
	case ModeStopwatch:
		return "Stopwatch, " + m.View() + ", " + state
	case ModeTimer:
		return "Timer, " + m.View() + " remaining, " + state
	default:
		return "Clock, " + m.View()
	}
}
