// Package scrollbar is a scrollbar widget: a track with a thumb whose size
// and position show how much of some content is visible and where. It does
// not own the content. The app gives it the content's Total size, how much
// is Visible and the current Offset (the numbers viewport, logview and
// virtuallist already keep), draws it beside the content, and applies the
// ScrolledMsg it sends back when the user scrolls it by key, wheel, click or
// drag.
//
// The bar is vertical by default (one column wide) or horizontal (one row
// tall). It has a LayoutNode, draws straight into a cell grid
// (layout.CellNode), and reads its colours from a theme.Theme.
package scrollbar

import (
	"math"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Orientation is the axis a scrollbar runs along.
type Orientation int

const (
	// Vertical is a one-column bar along the right edge of the content. It
	// is the zero value.
	Vertical Orientation = iota
	// Horizontal is a one-row bar along the bottom edge of the content.
	Horizontal
)

// WheelStep is how many content units one wheel notch scrolls when the
// Model's Wheel field is 0.
const WheelStep = 3

// Model is a scrollbar. Total, Visible and Offset are in the content's own
// units (lines or columns); the thumb is Visible/Total of the track long and
// sits at Offset/(Total-Visible) of the way along.
type Model struct {
	// Total is the size of the whole content, Visible how much of it shows
	// at once, and Offset where the visible window starts (0 to
	// Total-Visible).
	Total, Visible, Offset int
	// Orientation is Vertical (the zero value) or Horizontal.
	Orientation Orientation
	// Length is the track length in cells. When Bounds is not empty the
	// track is as long as Bounds on the bar's axis and Length is ignored.
	Length int
	// Wheel is the content units per wheel notch; 0 means WheelStep.
	Wheel int
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the key bindings Update obeys. New fills it with
	// DefaultKeyMap; a Model built as a literal with no bindings set behaves
	// as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// the thumb starts a drag, a press on the track pages toward it, and the
	// wheel over Bounds scrolls. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the bar; mouse
	// events outside it are ignored (a drag in progress keeps following the
	// pointer wherever it goes).
	Bounds hittest.Rect

	dragging bool
	grab     int // cells from the thumb's start where the drag began
}

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	Up       keymap.Binding
	Down     keymap.Binding
	PageUp   keymap.Binding
	PageDown keymap.Binding
	Start    keymap.Binding
	End      keymap.Binding
}

// DefaultKeyMap returns the default bindings: up/down (k/j) scroll one unit,
// pgup/pgdown a page, home/end jump to the ends. Left and right also scroll a
// Horizontal bar.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       keymap.NewBinding("scroll up", "up", "k", "left"),
		Down:     keymap.NewBinding("scroll down", "down", "j", "right"),
		PageUp:   keymap.NewBinding("page up", "pgup"),
		PageDown: keymap.NewBinding("page down", "pgdown"),
		Start:    keymap.NewBinding("scroll to start", "home"),
		End:      keymap.NewBinding("scroll to end", "end"),
	}
}

func (k KeyMap) list() []keymap.Binding {
	return []keymap.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Start, k.End}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	k := m.KeyMap
	n := 0
	for _, b := range k.list() {
		n += len(b.Keys)
	}
	if n == 0 {
		return DefaultKeyMap()
	}
	return k
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().list() }

// New builds a vertical scrollbar for content of total units with visible
// shown, at offset 0.
func New(total, visible int) Model {
	m := Model{Total: total, Visible: visible, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
	return m
}

// ScrolledMsg is delivered (via the Cmd Update returns) when the user moves
// the offset. The app applies Offset to its content.
type ScrolledMsg struct{ Offset int }

// MaxOffset is the largest Offset: Total-Visible, or 0 when everything is
// visible.
func (m Model) MaxOffset() int { return max(m.Total-max(m.Visible, 0), 0) }

// Scrollable reports whether there is content beyond the visible window.
func (m Model) Scrollable() bool { return m.MaxOffset() > 0 }

// SetOffset sets the offset to o, clamped to 0..MaxOffset.
func (m *Model) SetOffset(o int) { m.Offset = clamp(o, 0, m.MaxOffset()) }

// SetContent sets Total and Visible together and re-clamps the offset.
func (m *Model) SetContent(total, visible int) {
	m.Total, m.Visible = total, visible
	m.SetOffset(m.Offset)
}

// Percent is the scroll position from 0 (top) to 1 (end); it is 0 when
// nothing can scroll.
func (m Model) Percent() float64 {
	if mo := m.MaxOffset(); mo > 0 {
		return float64(clamp(m.Offset, 0, mo)) / float64(mo)
	}
	return 0
}

// track is the track length in cells.
func (m Model) track() int {
	if !m.Bounds.Empty() {
		if m.Orientation == Horizontal {
			return m.Bounds.W
		}
		return m.Bounds.H
	}
	return max(m.Length, 0)
}

// Thumb returns the thumb's start and size in cells along a track of length
// n: the size is n*Visible/Total (at least 1, at most n) and the start moves
// from 0 to n-size as Offset goes from 0 to MaxOffset. When nothing can
// scroll the thumb fills the track.
func (m Model) Thumb(n int) (start, size int) {
	if n <= 0 {
		return 0, 0
	}
	if !m.Scrollable() || m.Total <= 0 {
		return 0, n
	}
	size = clamp(int(math.Round(float64(n)*float64(max(m.Visible, 0))/float64(m.Total))), 1, n)
	start = int(math.Round(float64(n-size) * m.Percent()))
	return start, size
}

// offsetFor is the Offset that puts the thumb's start at cell start of a
// track of length n.
func (m Model) offsetFor(start, n int) int {
	_, size := m.Thumb(n)
	if span := n - size; span > 0 {
		return clamp(int(math.Round(float64(start)/float64(span)*float64(m.MaxOffset()))), 0, m.MaxOffset())
	}
	return m.Offset
}

// Dragging reports whether the thumb is being dragged.
func (m Model) Dragging() bool { return m.dragging }

// Update scrolls on the KeyMap bindings and, with Mouse on, on mouse events
// (see Model.Mouse). It returns a Cmd delivering ScrolledMsg when the offset
// changed. Any other Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	old := m.Offset
	switch ev := msg.(type) {
	case tui.MouseEvent:
		m = m.updateMouse(ev)
	case tui.Key:
		k := m.keys()
		page := max(m.Visible, 1)
		switch {
		case keymap.Matches(msg, k.Up):
			m.SetOffset(m.Offset - 1)
		case keymap.Matches(msg, k.Down):
			m.SetOffset(m.Offset + 1)
		case keymap.Matches(msg, k.PageUp):
			m.SetOffset(m.Offset - page)
		case keymap.Matches(msg, k.PageDown):
			m.SetOffset(m.Offset + page)
		case keymap.Matches(msg, k.Start):
			m.SetOffset(0)
		case keymap.Matches(msg, k.End):
			m.SetOffset(m.MaxOffset())
		}
	}
	if m.Offset == old {
		return m, nil
	}
	o := m.Offset
	return m, func() tui.Msg { return ScrolledMsg{Offset: o} }
}

func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse {
		return m
	}
	n := m.track()
	pos := ev.Y - m.Bounds.Y
	if m.Orientation == Horizontal {
		pos = ev.X - m.Bounds.X
	}
	switch {
	case m.dragging && ev.Action == tui.MouseActionMotion:
		m.SetOffset(m.offsetFor(pos-m.grab, n))
	case ev.Action == tui.MouseActionRelease && ev.Button != tui.MouseButtonWheelUp && ev.Button != tui.MouseButtonWheelDown:
		m.dragging = false
	case !m.Bounds.Contains(ev.X, ev.Y):
	case ev.Action != tui.MouseActionPress:
	case ev.Button == tui.MouseButtonWheelUp || ev.Button == tui.MouseButtonWheelLeft:
		m.SetOffset(m.Offset - m.wheel())
	case ev.Button == tui.MouseButtonWheelDown || ev.Button == tui.MouseButtonWheelRight:
		m.SetOffset(m.Offset + m.wheel())
	case ev.Button == tui.MouseButtonLeft && m.Scrollable():
		start, size := m.Thumb(n)
		switch {
		case pos < start:
			m.SetOffset(m.Offset - max(m.Visible, 1))
		case pos >= start+size:
			m.SetOffset(m.Offset + max(m.Visible, 1))
		default:
			m.dragging, m.grab = true, pos-start
		}
	}
	return m
}

func (m Model) wheel() int {
	if m.Wheel > 0 {
		return m.Wheel
	}
	return WheelStep
}

// glyphs returns the thumb and track glyphs: Theme.Glyphs.BarFull (solid) and
// BarEmpty (light), so the thumb reads without colour and an ASCII theme draws
// ASCII.
func (m Model) glyphs() (thumb, track string) {
	g := m.themed().Glyphs.Resolved()
	return g.BarFull, g.BarEmpty
}

func (m Model) styles() (thumb, track ansi.Style) {
	return ansi.NewStyle().Foreground(m.themed().Primary), ansi.NewStyle().Foreground(m.themed().Muted)
}

// cells returns the n track cells, each styled, in order.
func (m Model) cells(n int) []string {
	thumb, track := m.styles()
	thumbGlyph, trackGlyph := m.glyphs()
	start, size := m.Thumb(n)
	out := make([]string, n)
	for i := range out {
		if i >= start && i < start+size {
			out[i] = thumb.Render(thumbGlyph)
		} else {
			out[i] = track.Render(trackGlyph)
		}
	}
	return out
}

// View renders the bar over its track (Bounds, or Length when Bounds is
// empty): one column of rows for Vertical, one row for Horizontal.
func (m Model) View() string { return m.render(m.track()) }

func (m Model) render(n int) string {
	cs := m.cells(n)
	if m.Orientation == Horizontal {
		return strings.Join(cs, "")
	}
	return strings.Join(cs, "\n")
}

// Linearize describes the bar as plain text for accessible output (see
// tui.Linearizer), e.g. "Scrollbar, vertical, 25% scrolled, showing 10 of 40"
// or "Scrollbar, horizontal, all content visible".
func (m Model) Linearize() string {
	axis := "vertical"
	if m.Orientation == Horizontal {
		axis = "horizontal"
	}
	if !m.Scrollable() {
		return "Scrollbar, " + axis + ", all content visible"
	}
	pct := int(math.Round(m.Percent() * 100))
	return "Scrollbar, " + axis + ", " + strconv.Itoa(pct) + "% scrolled, showing " +
		strconv.Itoa(m.Visible) + " of " + strconv.Itoa(m.Total)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proofs that Model satisfies the widget contracts.
var (
	_ tui.Component[Model] = Model{}
	_ tui.Linearizer       = Model{}
)
