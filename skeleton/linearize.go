package skeleton

// Linearize renders the placeholder as plain text for accessible output (see
// tui.Linearizer): "Loading placeholder" for a visible skeleton, "" for one
// with no size. The shimmer animation is not spoken.
func (m Model) Linearize() string {
	if m.Width <= 0 || m.Lines <= 0 {
		return ""
	}
	return "Loading placeholder"
}
