package faces

// Linearize renders the face as plain text for accessible output (see
// tui.Linearizer): its name and what it is doing, e.g. "Face: Sleepy,
// sleep". The animation frames are decoration and are not spoken.
func (m Model) Linearize() string {
	f := m.Face()
	return "Face: " + f.Name + ", " + f.Anim
}
