// Package tooltip is a short hint shown next to the thing it describes
// while the pointer is over that thing. It is an overlay: Render draws it on
// top of a finished frame, under its target, or above when there is no room
// below.
//
// The pointer shows it only when the program reports motion with no button
// held, tui.WithMouse(tui.MouseAllMotion). Without a mouse, Show it when its
// target takes keyboard focus and Hide it when focus leaves.
//
// Stability: experimental. Its API may change in any minor release.
package tooltip

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Model is one tooltip. The zero value is closed and has no text; build one
// with New.
type Model struct {
	// Text is the hint. It may have several lines.
	Text string
	// Width, when above 0, wraps Text to that many cells. Zero or less
	// keeps each line of Text as it is.
	Width int
	// Target is the screen rectangle of the thing the tooltip describes.
	// The tooltip is drawn under it and lined up with its left edge.
	Target hittest.Rect
	Theme  theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens

	// Mouse, when true, makes Update handle tui.MouseEvent: the pointer
	// moving onto Target shows the tooltip, and the pointer leaving it, or
	// any button pressed, hides it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the base view the tooltip is
	// rendered over.
	Bounds hittest.Rect

	open bool
}

// New returns a closed tooltip with the given text for the thing drawn at
// target.
func New(text string, target hittest.Rect) Model {
	return Model{Text: text, Target: target, Theme: theme.DarkTheme()}
}

// Open reports whether the tooltip is shown.
func (m Model) Open() bool { return m.open }

// Show opens the tooltip.
func (m *Model) Show() { m.open = true }

// Hide closes the tooltip.
func (m *Model) Hide() { m.open = false }

// Update shows and hides the tooltip as the pointer moves onto and off
// Target, when Mouse is on. A button pressed anywhere hides it, and so does
// any key, since either means the user has gone on to something else. It
// never returns a Cmd and never keeps a message from the widgets behind it:
// forward the same message to them.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	switch ev := msg.(type) {
	case tui.MouseEvent:
		if !m.Mouse {
			return m, nil
		}
		m.open = ev.Action == tui.MouseActionMotion && ev.Button == tui.MouseButtonNone && m.Target.Contains(ev.X, ev.Y)
	case tui.Key:
		if ev.Action != tui.KeyRelease {
			m.open = false
		}
	}
	return m, nil
}

// extent returns s's rendered width (the widest line) and height (line
// count).
func extent(s string) (w, h int) {
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		w = max(w, ansi.Width(line))
	}
	return w, len(lines)
}

// place renders the tooltip's box and returns it with its offset and size
// inside a base of baseW x baseH whose top-left corner is Bounds'.
func (m Model) place(baseW, baseH int) (box string, x, y, bw, bh int) {
	t := m.themed()
	text := ansi.Sanitize(m.Text)
	if m.Width > 0 {
		text = ansi.Wrap(text, m.Width)
	}
	// A blank cell either side of each line, and no blank rows: a hint is
	// small.
	lines := strings.Split(text, "\n")
	style := ansi.NewStyle().Foreground(t.Text)
	for i, l := range lines {
		lines[i] = " " + style.Render(l) + " "
	}
	box = boxdraw.Draw(layout.NewBox().BorderColor(t.Info), t.Border, 0, 0, strings.Join(lines, "\n"))
	bw, bh = extent(box)
	x = max(min(m.Target.X-m.Bounds.X, baseW-bw), 0)
	top := m.Target.Y - m.Bounds.Y
	if y = top + m.Target.H; y+bh > baseH {
		y = top - bh
	}
	return box, x, max(y, 0), bw, bh
}

// Rect is the screen rectangle the tooltip takes when rendered over a base
// view filling Bounds. It is the zero Rect while the tooltip is closed.
func (m Model) Rect() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	_, x, y, w, h := m.place(m.Bounds.W, m.Bounds.H)
	return hittest.Rect{X: m.Bounds.X + x, Y: m.Bounds.Y + y, W: w, H: h}
}

// Render draws the tooltip over base: a bordered box in the theme's Info
// colour, its top-left corner under Target's bottom-left. A box that would
// pass base's right edge is moved left, and one that would pass its last
// row is drawn above Target. A closed tooltip returns base unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}
	baseW, baseH := extent(base)
	box, x, y, _, _ := m.place(baseW, baseH)
	return layout.Overlay(base, box, x, y)
}

// Compile-time proof that Model satisfies tui.Overlay[Model] — see
// tui.Overlay's doc comment for what this contract means and why.
var _ tui.Overlay[Model] = Model{}

// LayoutNode returns the tooltip's box as an ordinary layout.Node, for an
// app that places it itself. While the tooltip is closed it measures as
// nothing. The Model is not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	box, _, _, _, _ := m.place(0, 0)
	return layout.Block(box)
}

// Linearize renders the tooltip as plain text for accessible output (see
// tui.Linearizer): "Hint: " and its text, or "" while it is closed.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	return "Hint: " + ansi.Sanitize(m.Text)
}
