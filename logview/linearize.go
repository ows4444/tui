package logview

// Linearize renders the log as plain text for accessible output (see
// tui.Linearizer): every line, oldest first, with escape sequences removed.
// There is no header, so an appended line adds only one new line to an
// append-only transcript.
func (m Model) Linearize() string { return m.Viewport.Linearize() }
