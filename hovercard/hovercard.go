// Package hovercard is a card of details shown next to the thing it
// describes while the pointer is over that thing: a preview of a link, a
// profile behind a name. It is an overlay: Render draws it on top of a
// finished frame, under its target, or above when there is no room below.
//
// It differs from tooltip in what it holds and how long it stays. It has a
// title and a body the caller has drawn, and it stays open while the pointer
// is on the card itself, so the pointer can move from the target onto it.
//
// The pointer opens it only when the program reports motion with no button
// held, tui.WithMouse(tui.MouseAllMotion). Without a mouse, Show it from a
// key of the app's own and let Esc close it.
//
// Stability: experimental. Its API may change in any minor release.
package hovercard

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Model is one hover card. The zero value is closed and empty; build one
// with New.
type Model struct {
	// ID is carried by ClosedMsg, to tell one card's closing from another's.
	ID string
	// Title heads the card, in bold. Empty leaves the heading out.
	Title string
	// Content is the body: text the caller has already drawn, of any number
	// of lines. Its colours and attributes are kept and every other escape
	// sequence is dropped.
	Content string
	// Width, when above 0, wraps Content to that many cells. Zero or less
	// keeps each line as it is.
	Width int
	// Target is the screen rectangle of the thing the card describes. The
	// card is drawn under it and lined up with its left edge.
	Target hittest.Rect
	Theme  theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys that close the card. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: the pointer
	// moving onto Target opens the card; moving off both Target and the
	// card, or a press outside both, closes it. Off (the default) ignores
	// the mouse.
	Mouse bool
	// Bounds is the screen rectangle of the base view the card is rendered
	// over.
	Bounds hittest.Rect

	open bool
}

// New returns a closed card with the given title and content for the thing
// drawn at target.
func New(title, content string, target hittest.Rect) Model {
	return Model{Title: title, Content: content, Target: target, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Close keymap.Binding // close the card and deliver ClosedMsg
}

// DefaultKeyMap returns Esc.
func DefaultKeyMap() KeyMap {
	return KeyMap{Close: keymap.NewBinding("close", "esc")}
}

func (m Model) keys() KeyMap {
	if len(m.KeyMap.Close.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the actions the card honours now, with descriptions, for
// help text. A closed card honours none.
func (m Model) Bindings() []keymap.Binding {
	if !m.open {
		return nil
	}
	return []keymap.Binding{m.keys().Close}
}

// ClosedMsg is delivered (via the Cmd Update returns) when the card closes
// by the pointer or by Esc. Hide delivers nothing.
type ClosedMsg struct {
	// ID is the card's ID.
	ID string
}

// Open reports whether the card is shown.
func (m Model) Open() bool { return m.open }

// Show opens the card.
func (m *Model) Show() { m.open = true }

// Hide closes the card. No ClosedMsg is delivered.
func (m *Model) Hide() { m.open = false }

func (m Model) shut() (Model, tui.Cmd) {
	m.open = false
	msg := ClosedMsg{ID: m.ID}
	return m, func() tui.Msg { return msg }
}

// Update opens the card when the pointer moves onto Target and closes it
// when the pointer moves off both Target and the card, or a button is
// pressed outside both, when Mouse is on. Esc closes an open card. It does
// not keep a message from the widgets behind it: forward the same message
// to them.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		if !m.Mouse {
			return m, nil
		}
		onTarget := m.Target.Contains(ev.X, ev.Y)
		hover := ev.Action == tui.MouseActionMotion && ev.Button == tui.MouseButtonNone
		switch {
		case !m.open && hover && onTarget:
			m.open = true
		case m.open && !onTarget && !m.Rect().Contains(ev.X, ev.Y) && (hover || ev.Action == tui.MouseActionPress):
			return m.shut()
		}
		return m, nil
	}
	if m.open && keymap.Matches(msg, m.keys().Close) {
		return m.shut()
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

// body is the card's text: the title, a blank row and the content, either
// left out when empty.
func (m Model) body() string {
	t := m.themed()
	var parts []string
	if title := ansi.Sanitize(m.Title); title != "" {
		parts = append(parts, ansi.NewStyle().Bold().Foreground(t.Primary).Render(title))
	}
	if content := ansi.SanitizeKeepSGR(ansi.ExpandTabs(m.Content)); content != "" {
		if m.Width > 0 {
			content = ansi.WrapStyled(content, m.Width)
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n")
}

// place renders the card and returns it with its offset and size inside a
// base of baseW x baseH whose top-left corner is Bounds'.
func (m Model) place(baseW, baseH int) (box string, x, y, bw, bh int) {
	t := m.themed()
	// A blank cell either side of each line, and no blank rows above or
	// below: a terminal has few rows to spend.
	lines := strings.Split(m.body(), "\n")
	for i, l := range lines {
		lines[i] = " " + l + " "
	}
	box = boxdraw.Draw(layout.NewBox().BorderColor(t.BorderColor), t.Border, 0, 0, strings.Join(lines, "\n"))
	bw, bh = extent(box)
	x = max(min(m.Target.X-m.Bounds.X, baseW-bw), 0)
	top := m.Target.Y - m.Bounds.Y
	if y = top + m.Target.H; y+bh > baseH {
		y = top - bh
	}
	return box, x, max(y, 0), bw, bh
}

// Rect is the screen rectangle the card takes when rendered over a base
// view filling Bounds. It is the zero Rect while the card is closed.
func (m Model) Rect() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	_, x, y, w, h := m.place(m.Bounds.W, m.Bounds.H)
	return hittest.Rect{X: m.Bounds.X + x, Y: m.Bounds.Y + y, W: w, H: h}
}

// Render draws the card over base: a bordered box, its top-left corner
// under Target's bottom-left, so the pointer can move from one onto the
// other without crossing a gap. A card that would pass base's right edge is
// moved left, and one that would pass its last row is drawn above Target. A
// closed card returns base unchanged.
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

// LayoutNode returns the card as an ordinary layout.Node, for an app that
// places it itself. While the card is closed it measures as nothing. The
// Model is not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	box, _, _, _, _ := m.place(0, 0)
	return layout.Block(box)
}

// Linearize renders the card as plain text for accessible output (see
// tui.Linearizer): its title and its content without styling, or "" while
// it is closed.
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	return ansi.StripANSI(m.body())
}
