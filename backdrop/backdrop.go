// Package backdrop dims a finished frame, so that what is drawn over it
// next, a dialog or a drawer, stands out and the rest reads as out of
// reach. It is an overlay: Render takes the frame and returns it dimmed.
//
//	frame = dialog.Render(backdrop.Render(view))
//
// Stability: experimental. Its API may change in any minor release.
package backdrop

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Model is one backdrop. The zero value is closed; build one with New.
type Model struct {
	// ID is carried by PressedMsg, to tell one backdrop's press from
	// another's.
	ID    string
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens

	// Mouse, when true, makes Update handle tui.MouseEvent while the
	// backdrop is open: a left press on it delivers PressedMsg. Off (the
	// default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the frame the backdrop dims.
	Bounds hittest.Rect
	// Hole is the screen rectangle of what is drawn over the backdrop. A
	// press inside it is not a press on the backdrop.
	Hole hittest.Rect

	open bool
}

// New returns an open backdrop.
func New() Model { return Model{Theme: theme.DarkTheme(), open: true} }

// Open reports whether the backdrop is shown.
func (m Model) Open() bool { return m.open }

// Show opens the backdrop.
func (m *Model) Show() { m.open = true }

// Hide closes the backdrop.
func (m *Model) Hide() { m.open = false }

// PressedMsg is delivered (via the Cmd Update returns) when the left button
// is pressed on an open backdrop, outside Hole. The backdrop stays open:
// the app decides whether a press outside closes what is in front.
type PressedMsg struct {
	// ID is the backdrop's ID.
	ID string
}

// Update reports a left press on the backdrop, when it is open and Mouse is
// on. It ignores every other message.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	ev, ok := msg.(tui.MouseEvent)
	if !ok || !m.open || !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft {
		return m, nil
	}
	if !m.Bounds.Contains(ev.X, ev.Y) || m.Hole.Contains(ev.X, ev.Y) {
		return m, nil
	}
	pressed := PressedMsg{ID: m.ID}
	return m, func() tui.Msg { return pressed }
}

// Render returns base with its colours and attributes taken off and every
// row drawn faint in the theme's Muted colour. Rows keep their width. A
// closed backdrop returns base unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}
	dim := ansi.NewStyle().Faint().Foreground(m.themed().Muted)
	lines := strings.Split(ansi.Sanitize(base), "\n")
	for i, l := range lines {
		if l != "" {
			lines[i] = dim.Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}

// LayoutNode returns a node that draws nothing: a backdrop has no content
// of its own, only what Render does to a frame. It is here so a backdrop
// can stand where any component does. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return layout.Block("") }

// Linearize returns "": a backdrop has nothing to read out. It is here so
// a backdrop can stand where any component does.
func (m Model) Linearize() string { return "" }
