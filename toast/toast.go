// Package toast is a transient auto-dismissing notification — InkUI's
// "Toast": a small bordered box in a screen corner, composited via
// layout.Overlay the same way dialog is, that closes itself after
// Duration via motion.After rather than waiting for a keypress.
package toast

import (
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Position selects which screen corner a Toast appears in.
type Position int

const (
	// BottomRight places the toast in the bottom-right corner, the zero
	// value and so the default.
	BottomRight Position = iota
	// BottomLeft places the toast in the bottom-left corner.
	BottomLeft
	// TopRight places the toast in the top-right corner.
	TopRight
	// TopLeft places the toast in the top-left corner.
	TopLeft
)

// Model is a corner notification. Like dialog.Model, Render composites it
// over a base view when open and returns base unchanged when closed, so a
// parent can call it unconditionally every frame.
type Model struct {
	Message string

	// Raw, when true, draws Message unchanged. By default it is sanitised
	// (ansi.Sanitize) so untrusted text cannot carry terminal escape
	// sequences.
	Raw     bool
	Variant widgets.Variant
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens   theme.Tokens
	Duration time.Duration
	Position Position
	// Margin insets the toast from base's edge, in cells. Defaults to 1
	// via New so a corner toast doesn't sit exactly flush against (and
	// visually merge into) whatever border base itself might have right
	// at that edge — set to 0 for a flush-to-the-corner placement.
	Margin int

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on
	// the toast dismisses it at once and delivers DismissedMsg. Off (the
	// default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the base view the toast is rendered
	// over (its origin and size, as passed to Render); the toast's own
	// rectangle is worked out from it, Position and Margin. Events outside
	// the toast are ignored.
	Bounds hittest.Rect

	open bool
	// id disambiguates Show calls: each one starts a new dismiss timer
	// carrying the id current at the time. If Show is called again before
	// a previous timer fires, that stale timer's dismissMsg carries an id
	// that no longer matches m.id, so Update ignores it instead of closing
	// the new toast early.
	id int
}

// New returns a Model that isn't shown yet — call Show (and return its
// Cmd) to actually display it.
func New(message string) Model {
	return Model{
		Message:  message,
		Theme:    theme.DarkTheme(),
		Variant:  widgets.VariantInfo,
		Duration: 3 * time.Second,
		Position: BottomRight,
		Margin:   1,
	}
}

// Open reports whether the toast is currently shown.
func (m Model) Open() bool { return m.open }

// Show opens the toast and returns a Cmd that auto-dismisses it after
// Duration — return this from your own Update (or Init) so it actually
// fires.
func (m *Model) Show() tui.Cmd {
	m.open = true
	m.id++
	id := m.id
	return tui.FromCtx(motion.After(m.Duration, func(time.Time) tui.Msg { return dismissMsg{id: id} }))
}

// Hide closes the toast immediately, without waiting for Duration.
func (m *Model) Hide() { m.open = false }

type dismissMsg struct{ id int }

// DismissedMsg is delivered (via the Cmd Update returns) when the toast
// auto-dismisses after Duration.
type DismissedMsg struct{}

// Update closes the toast when its own auto-dismiss timer (from Show)
// fires, delivering DismissedMsg; a no-op for any other Msg or a stale
// timer from a toast Show has since replaced. With Mouse on, a left click on
// the toast also closes it and delivers DismissedMsg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && m.open && ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress &&
			m.Rect().Contains(ev.X, ev.Y) {
			m.open = false
			return m, func() tui.Msg { return DismissedMsg{} }
		}
		return m, nil
	}
	dm, ok := msg.(dismissMsg)
	if !ok || dm.id != m.id || !m.open {
		return m, nil
	}
	m.open = false
	return m, func() tui.Msg { return DismissedMsg{} }
}

// build renders the toast box and returns it with its size.
func (m Model) build() (box string, w, h int) {
	color := m.Variant.Color(m.themed())
	text := ansi.Clean(m.Raw, m.Message)
	// The level shows without colour too. VariantInfo, New's default, keeps
	// the plain message; every other level leads with its icon.
	if m.Variant != widgets.VariantInfo {
		text = m.Variant.Icon(m.themed().GlyphSet()) + " " + text
	}
	msgStyled := ansi.NewStyle().Foreground(color).Render(text)
	box = boxdraw.Draw(layout.NewBox().BorderColor(m.themed().BorderColor), m.themed().Border, m.themed().ResolvedSpacing().S, 0, msgStyled)
	lines := strings.Split(box, "\n")
	return box, ansi.Width(lines[0]), len(lines)
}

// place is the toast's offset inside a base of baseWidth x baseHeight.
func (m Model) place(boxWidth, boxHeight, baseWidth, baseHeight int) (x, y int) {
	switch m.Position {
	case TopLeft:
		x, y = m.Margin, m.Margin
	case TopRight:
		x, y = baseWidth-boxWidth-m.Margin, m.Margin
	case BottomLeft:
		x, y = m.Margin, baseHeight-boxHeight-m.Margin
	default: // BottomRight
		x, y = baseWidth-boxWidth-m.Margin, baseHeight-boxHeight-m.Margin
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// Rect is the screen rectangle the toast occupies when rendered over a base
// view filling Bounds: where a click lands on it. It is the zero Rect while
// the toast is closed.
func (m Model) Rect() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	_, w, h := m.build()
	x, y := m.place(w, h, m.Bounds.W, m.Bounds.H)
	return hittest.Rect{X: m.Bounds.X + x, Y: m.Bounds.Y + y, W: w, H: h}
}

// Render composites the toast over base in its configured Position. If
// closed, it returns base unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}

	box, boxWidth, boxHeight := m.build()

	baseLines := strings.Split(base, "\n")
	baseWidth := 0
	for _, l := range baseLines {
		if w := ansi.Width(l); w > baseWidth {
			baseWidth = w
		}
	}
	baseHeight := len(baseLines)

	x, y := m.place(boxWidth, boxHeight, baseWidth, baseHeight)
	return layout.Overlay(base, box, x, y)
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}
