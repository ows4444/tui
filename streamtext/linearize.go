package streamtext

import "github.com/ows4444/tui/ansi"

// Linearize renders the text as plain text for accessible output (see
// tui.Linearizer): all of it, wrapped as View would wrap it, without waiting
// for the typewriter reveal or drawing its cursor. A reader gets the text
// at once instead of character by character.
func (m Model) Linearize() string { return ansi.StripANSI(m.display()) }
