package popover

// Linearize renders the popover as plain text for accessible output (see
// tui.Linearizer): "Popover" and then its content, or "" while it is closed.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	if m.Content == "" {
		return "Popover"
	}
	return "Popover\n" + m.Content
}
