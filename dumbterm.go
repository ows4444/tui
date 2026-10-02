package tui

import "os"

// applyDumbTermDefaults makes TERM=dumb run without the alternate screen,
// cursor movement, or synchronized output (mode 2026), unless the caller
// explicitly chose a mode with WithAltScreen or WithAccessible. It reuses
// the accessible (append-only) renderer, which emits none of those
// sequences. NewProgram calls it after every option has been applied.
func (p *Program) applyDumbTermDefaults() {
	if p.modeSet || os.Getenv("TERM") != "dumb" {
		return
	}
	p.accessible = true
}
