package clipboard

// Linearize renders the button as plain text for accessible output (see
// tui.Linearizer): "<label>, button", or "Copied to clipboard" while the
// confirmation is showing. The text being copied is not spoken.
func (m Model) Linearize() string {
	if m.copied {
		return "Copied to clipboard"
	}
	return m.Label + ", button"
}
