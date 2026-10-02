package dialog

import "github.com/ows4444/tui/ansi"

// Linearize renders the dialog as plain text for accessible output (see
// tui.Linearizer): "Dialog: <title>" and then the message, or "" while it is
// closed. An overlay has no base to composite over in a linear transcript, so
// an app that uses accessible mode shows this in addition to its own text.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	out := "Dialog: " + ansi.Clean(m.Raw, m.Title)
	if m.Message != "" {
		out += "\n" + ansi.Clean(m.Raw, m.Message)
	}
	return out
}
