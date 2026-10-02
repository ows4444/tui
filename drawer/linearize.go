package drawer

// Linearize renders the drawer as plain text for accessible output (see
// tui.Linearizer): "Drawer, <edge> edge" and then its content, or "" while
// it is closed.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	edge := map[Edge]string{EdgeRight: "right", EdgeLeft: "left", EdgeTop: "top", EdgeBottom: "bottom"}[m.Edge]
	out := "Drawer, " + edge + " edge"
	if m.Content != "" {
		out += "\n" + m.Content
	}
	return out
}
