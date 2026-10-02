// Package splitpane is a two-pane split with a draggable divider: two
// layout.Nodes side by side (Columns) or stacked (Rows), separated by a
// one-cell divider that the user moves with the keyboard or the mouse. The
// panes never shrink below their minimum sizes.
//
// The Model owns only the divider position. The panes are layout.Nodes the
// app supplies (any widget's LayoutNode), drawn by the split's own LayoutNode,
// which also draws into a cell grid (layout.CellNode). For the mouse, tell the
// Model where it is drawn with Bounds (see layout.RectOf) and turn Mouse on.
package splitpane

import (
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Direction is how the two panes are arranged.
type Direction int

const (
	// Columns puts the panes side by side with a vertical divider; Pos is
	// the first pane's width. It is the zero value.
	Columns Direction = iota
	// Rows stacks the panes with a horizontal divider; Pos is the first
	// pane's height.
	Rows
)

// Model is a split pane. Pos, the size of the first pane, is kept clamped so
// that the first pane is at least Min1 and the second at least Min2 cells
// along the split axis (when both cannot fit, Min1 wins).
type Model struct {
	// First and Second are the panes: left/top and right/bottom.
	First, Second layout.Node
	// Direction is Columns (the zero value) or Rows.
	Direction Direction
	// Min1 and Min2 are the smallest sizes of the first and second pane
	// along the split axis, in cells.
	Min1, Min2 int
	// Step is the cells one Grow or Shrink key moves the divider; 0 means 1.
	Step int
	// Total is the size along the split axis, divider included, used when
	// Bounds is empty. LayoutNode ignores it and uses the Size it is given.
	Total int
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the key bindings Update obeys. New fills it with
	// DefaultKeyMap; a Model built as a literal with no bindings set behaves
	// as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: pressing the
	// divider and dragging moves it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the split; the
	// divider is hit-tested within it, and its extent along the split axis
	// is the Total.
	Bounds hittest.Rect

	pos      int
	set      bool
	dragging bool
}

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	// Shrink moves the divider toward the first pane (left or up).
	Shrink keymap.Binding
	// Grow moves the divider toward the second pane (right or down).
	Grow keymap.Binding
	// Reset puts the divider back in the middle.
	Reset keymap.Binding
}

// DefaultKeyMap returns the default bindings: ctrl+left or ctrl+up shrinks
// the first pane, ctrl+right or ctrl+down grows it, ctrl+e centres the
// divider.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Shrink: keymap.NewBinding("shrink first pane", "ctrl+left", "ctrl+up"),
		Grow:   keymap.NewBinding("grow first pane", "ctrl+right", "ctrl+down"),
		Reset:  keymap.NewBinding("centre divider", "ctrl+e"),
	}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	k := m.KeyMap
	if len(k.Shrink.Keys)+len(k.Grow.Keys)+len(k.Reset.Keys) == 0 {
		return DefaultKeyMap()
	}
	return k
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.Shrink, k.Grow, k.Reset}
}

// New builds a split of first and second, side by side, with the divider in
// the middle.
func New(first, second layout.Node) Model {
	return Model{First: first, Second: second, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// ChangedMsg is delivered (via the Cmd Update returns) when the divider
// moves. Pos is the new size of the first pane.
type ChangedMsg struct{ Pos int }

// total is the size along the split axis the divider is clamped to.
func (m Model) total() int {
	if !m.Bounds.Empty() {
		if m.Direction == Rows {
			return m.Bounds.H
		}
		return m.Bounds.W
	}
	return max(m.Total, 0)
}

// clampPos limits p to what total cells allow: Min1 <= p <= total-1-Min2,
// and never beyond the divider's own cell.
func (m Model) clampPos(p, total int) int {
	cap := max(total-1, 0)
	lo := min(max(m.Min1, 0), cap)
	hi := max(lo, min(total-1-max(m.Min2, 0), cap))
	return max(lo, min(p, hi))
}

// Pos returns the first pane's size in cells, clamped to the minimums.
func (m Model) Pos() int { return m.posFor(m.total()) }

func (m Model) posFor(total int) int {
	p := m.pos
	if !m.set {
		p = (total - 1) / 2
	}
	return m.clampPos(p, total)
}

// Sizes returns the first and second pane sizes along the split axis; with
// the one-cell divider they add up to the total.
func (m Model) Sizes() (first, second int) {
	t := m.total()
	p := m.posFor(t)
	return p, max(t-1-p, 0)
}

// SetPos puts the divider at p cells from the start, clamped to the minimums.
func (m *Model) SetPos(p int) {
	m.pos, m.set = m.clampPos(p, m.total()), true
}

// SetTotal sets Total and re-clamps the divider to it.
func (m *Model) SetTotal(n int) {
	m.Total = n
	m.SetPos(m.Pos())
}

// Dragging reports whether the divider is being dragged.
func (m Model) Dragging() bool { return m.dragging }

// DividerRect is the screen rectangle of the divider (one cell wide or tall)
// within Bounds, or the zero Rect when Bounds is empty.
func (m Model) DividerRect() hittest.Rect {
	if m.Bounds.Empty() {
		return hittest.Rect{}
	}
	p := m.Pos()
	if m.Direction == Rows {
		return hittest.Rect{X: m.Bounds.X, Y: m.Bounds.Y + p, W: m.Bounds.W, H: 1}
	}
	return hittest.Rect{X: m.Bounds.X + p, Y: m.Bounds.Y, W: 1, H: m.Bounds.H}
}

// Update moves the divider by Step on the Shrink and Grow bindings and
// centres it on Reset. With Mouse on, pressing the divider with the left
// button and dragging moves it; releasing drops it. The divider always stays
// within the minimums. It returns a Cmd delivering ChangedMsg when the
// position changed. Any other Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	old := m.Pos()
	switch ev := msg.(type) {
	case tui.MouseEvent:
		m = m.updateMouse(ev)
	case tui.Key:
		k := m.keys()
		step := max(m.Step, 1)
		switch {
		case keymap.Matches(msg, k.Shrink):
			m.SetPos(old - step)
		case keymap.Matches(msg, k.Grow):
			m.SetPos(old + step)
		case keymap.Matches(msg, k.Reset):
			m.SetPos((m.total() - 1) / 2)
		}
	}
	if p := m.Pos(); p != old {
		return m, func() tui.Msg { return ChangedMsg{Pos: p} }
	}
	return m, nil
}

func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse || m.Bounds.Empty() {
		return m
	}
	switch {
	case m.dragging && ev.Action == tui.MouseActionMotion:
		lx, ly := m.Bounds.Local(ev.X, ev.Y)
		if m.Direction == Rows {
			m.SetPos(ly)
		} else {
			m.SetPos(lx)
		}
	case ev.Action == tui.MouseActionRelease:
		m.dragging = false
	case ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft &&
		m.DividerRect().Contains(ev.X, ev.Y):
		m.dragging = true
	}
	return m
}

// View renders the split at Bounds' size (or Total by the Bounds' other
// extent when Bounds is empty, one cell across). Use LayoutNode to place it
// in a layout.
func (m Model) View() string {
	s := layout.Size{W: m.Bounds.W, H: m.Bounds.H}
	if m.Bounds.Empty() {
		if m.Direction == Rows {
			s = layout.Size{W: 1, H: m.Total}
		} else {
			s = layout.Size{W: m.Total, H: 1}
		}
	}
	return m.LayoutNode().Render(s)
}

// Linearize describes the split as plain text for accessible output (see
// tui.Linearizer): its arrangement and divider position, then each pane's own
// linear text when it has one, e.g. "Split pane, 2 columns, divider at 30 of
// 80" followed by "Pane 1:" and "Pane 2:" sections.
func (m Model) Linearize() string {
	axis := "columns"
	if m.Direction == Rows {
		axis = "rows"
	}
	out := "Split pane, 2 " + axis + ", divider at " + strconv.Itoa(m.Pos()) + " of " + strconv.Itoa(m.total())
	for i, n := range []layout.Node{m.First, m.Second} {
		out += "\nPane " + strconv.Itoa(i+1) + ":"
		if l, ok := n.(tui.Linearizer); ok {
			out += "\n" + l.Linearize()
		}
	}
	return out
}

// Compile-time proofs that Model satisfies the widget contracts.
var (
	_ tui.Component[Model] = Model{}
	_ tui.Linearizer       = Model{}
)
