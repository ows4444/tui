// Package window is a desktop of overlapping windows the user moves and
// resizes. Each Window has a rectangle on the desktop, a title bar with a close
// box, and a layout.Node for its content; the last window in Windows is on top
// and has focus. Dragging the title bar moves a window, dragging a side or the
// bottom edge resizes it, a press raises and focuses it, and the close box
// removes it. The keyboard does the same: Next and Prev change focus, Move and
// Resize keys act on the focused window, Maximize toggles it, Close removes it.
//
// The Model owns the rectangles and the order. The content of a window is a
// layout.Node the app sets on Window.Content each frame (any widget's
// LayoutNode); the app also routes keys, and mouse events that land in a body
// (see Hit and ContentRect), to the widget inside the focused window. For the
// mouse, tell the Model where the desktop is drawn with Bounds (see
// layout.RectOf) and turn Mouse on.
//
// Stability: experimental. Its API may change in any minor release.
package window

import (
	"slices"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Default minimum size of a window, border included.
const (
	defaultMinW = 10
	defaultMinH = 3
)

// Window is one window on the desktop.
type Window struct {
	// ID names the window in the messages and in the lookups. IDs should be
	// unique; the first window with an ID is the one a lookup finds.
	ID string
	// Title is drawn in the title bar. It is untrusted text: escape sequences
	// and control characters are stripped.
	Title string
	// Rect is the window with its border, in cells from the desktop's top-left.
	Rect hittest.Rect
	// MinW and MinH are the smallest size the user can make it, border
	// included. Values under 10 and 3 are read as 10 and 3.
	MinW, MinH int
	// Content is drawn inside the border, in (Rect.W-2) x (Rect.H-2) cells.
	Content layout.Node
	// Fixed stops the user moving, resizing or maximizing the window.
	Fixed bool
	// NoClose takes the close box away and ignores the Close key.
	NoClose bool

	restore   hittest.Rect // the Rect before Maximize
	maximized bool
}

// Maximized reports whether the window fills the desktop.
func (w Window) Maximized() bool { return w.maximized }

// Part is the part of a window a cell belongs to.
type Part int

// The parts Hit tells apart.
const (
	// PartNone is a cell no window covers.
	PartNone Part = iota
	// PartTitle is the title bar, including the top corners.
	PartTitle
	// PartClose is the close box in the title bar.
	PartClose
	// PartEdge is the left, right or bottom border, where a drag resizes.
	PartEdge
	// PartBody is inside the border.
	PartBody
)

// Model is a desktop of windows.
type Model struct {
	// Windows are the windows from back to front: the last is on top and has
	// focus. Use Add, Remove, Raise and SetRect to change them; a program that
	// edits the slice itself should treat it as owned by the Model afterwards.
	Windows []Window
	// Background is drawn behind the windows; nil leaves the desktop blank.
	Background layout.Node
	// Step is the cells one Move or Resize key moves a window; 0 means 1.
	Step  int
	Theme theme.Theme

	tokens theme.Tokens
	// KeyMap holds the key bindings Update obeys. New fills it with
	// DefaultKeyMap; a Model built as a literal with none set behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: see the package
	// comment. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the desktop. Windows
	// are kept inside it; with Bounds empty they are not clamped.
	Bounds hittest.Rect

	drag drag
}

// drag is a window being moved or resized with the mouse.
type drag struct {
	on                bool
	id                string
	move              bool         // moving; else resizing
	offX, offY        int          // where in the window a move holds it
	left, right, down bool         // the edges a resize holds
	start             hittest.Rect // the window's Rect when the drag began
}

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	// Next and Prev change the focused window: Next sends the top window to
	// the back, Prev brings the bottom one to the front.
	Next, Prev keymap.Binding
	// Close removes the focused window.
	Close keymap.Binding
	// Maximize toggles the focused window between its size and the desktop.
	Maximize keymap.Binding
	// MoveLeft, MoveRight, MoveUp and MoveDown move the focused window.
	MoveLeft, MoveRight, MoveUp, MoveDown keymap.Binding
	// ShrinkW and GrowW change its width, ShrinkH and GrowH its height.
	ShrinkW, GrowW, ShrinkH, GrowH keymap.Binding
}

// DefaultKeyMap returns the default bindings: alt+] and alt+[ change focus,
// alt+w closes, alt+m maximizes, alt and an arrow moves, alt+shift and an arrow
// resizes.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Next:      keymap.NewBinding("next window", "alt+]"),
		Prev:      keymap.NewBinding("previous window", "alt+["),
		Close:     keymap.NewBinding("close window", "alt+w"),
		Maximize:  keymap.NewBinding("maximize window", "alt+m"),
		MoveLeft:  keymap.NewBinding("move window left", "alt+left"),
		MoveRight: keymap.NewBinding("move window right", "alt+right"),
		MoveUp:    keymap.NewBinding("move window up", "alt+up"),
		MoveDown:  keymap.NewBinding("move window down", "alt+down"),
		ShrinkW:   keymap.NewBinding("narrower window", "alt+shift+left"),
		GrowW:     keymap.NewBinding("wider window", "alt+shift+right"),
		ShrinkH:   keymap.NewBinding("shorter window", "alt+shift+up"),
		GrowH:     keymap.NewBinding("taller window", "alt+shift+down"),
	}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	if len(m.KeyMap.Next.Keys)+len(m.KeyMap.Close.Keys)+len(m.KeyMap.MoveLeft.Keys)+len(m.KeyMap.GrowW.Keys) == 0 {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.Next, k.Prev, k.Close, k.Maximize, k.MoveLeft, k.MoveRight, k.MoveUp, k.MoveDown,
		k.ShrinkW, k.GrowW, k.ShrinkH, k.GrowH}
}

// New returns a desktop of windows, the last on top.
func New(windows ...Window) Model {
	return Model{Windows: slices.Clone(windows), Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// ChangedMsg is delivered (via the Cmd Update returns) when a window has moved
// or been resized: at the end of a drag, or at once for a key.
type ChangedMsg struct {
	ID   string
	Rect hittest.Rect
}

// ClosedMsg is delivered when a window was closed and removed.
type ClosedMsg struct{ ID string }

// FocusedMsg is delivered when a different window came to the front.
type FocusedMsg struct{ ID string }

func (m Model) index(id string) int {
	return slices.IndexFunc(m.Windows, func(w Window) bool { return w.ID == id })
}

// Focused returns the ID of the window on top, or "" with no windows.
func (m Model) Focused() string {
	if len(m.Windows) == 0 {
		return ""
	}
	return m.Windows[len(m.Windows)-1].ID
}

// Window returns the window with the ID.
func (m Model) Window(id string) (Window, bool) {
	if i := m.index(id); i >= 0 {
		return m.Windows[i], true
	}
	return Window{}, false
}

// own makes the Windows slice private to m, so changing it leaves the Models
// it was copied from alone.
func (m *Model) own() { m.Windows = slices.Clone(m.Windows) }

// Add puts w on top, clamped to the desktop.
func (m *Model) Add(w Window) {
	m.own()
	w.Rect = m.clamp(w, w.Rect)
	m.Windows = append(m.Windows, w)
}

// Remove takes the window out and reports whether there was one.
func (m *Model) Remove(id string) bool {
	i := m.index(id)
	if i < 0 {
		return false
	}
	m.own()
	m.Windows = slices.Delete(m.Windows, i, i+1)
	return true
}

// Raise brings the window to the front and reports whether there was one.
func (m *Model) Raise(id string) bool {
	i := m.index(id)
	if i < 0 {
		return false
	}
	m.own()
	w := m.Windows[i]
	m.Windows = append(slices.Delete(m.Windows, i, i+1), w)
	return true
}

// SetRect puts the window at r, clamped to its minimum size and the desktop.
func (m *Model) SetRect(id string, r hittest.Rect) {
	if i := m.index(id); i >= 0 {
		m.own()
		m.Windows[i].Rect = m.clamp(m.Windows[i], r)
		m.Windows[i].maximized = false
	}
}

// size is the desktop's size, and whether it is known.
func (m Model) size() (w, h int, ok bool) {
	return m.Bounds.W, m.Bounds.H, !m.Bounds.Empty()
}

// clamp limits r to w's minimum size and, when the desktop is known, to the
// desktop.
func (m Model) clamp(w Window, r hittest.Rect) hittest.Rect {
	r.W, r.H = max(r.W, w.MinW, defaultMinW), max(r.H, w.MinH, defaultMinH)
	if dw, dh, ok := m.size(); ok {
		r.W, r.H = max(min(r.W, dw), defaultMinW), max(min(r.H, dh), defaultMinH)
		r.X = max(min(r.X, dw-r.W), 0)
		r.Y = max(min(r.Y, dh-r.H), 0)
	}
	return r
}

// parts finds the part of window w a cell (rx, ry) from its top-left belongs
// to, and the edges of the border it is on.
func parts(w Window, rx, ry int) (p Part, left, right, down bool) {
	r := w.Rect
	switch {
	case ry == 0 && !w.NoClose && r.W >= defaultMinW && rx >= r.W-4 && rx < r.W-1:
		return PartClose, false, false, false
	case ry == 0:
		return PartTitle, false, false, false
	case rx == 0 || rx == r.W-1 || ry == r.H-1:
		return PartEdge, rx == 0, rx == r.W-1, ry == r.H-1
	}
	return PartBody, false, false, false
}

// Hit returns the window at screen cell (x, y), and the part of it. Windows
// are tested from the top.
func (m Model) Hit(x, y int) (id string, part Part) {
	lx, ly := m.Bounds.Local(x, y)
	for i := len(m.Windows) - 1; i >= 0; i-- {
		w := m.Windows[i]
		if w.Rect.Contains(lx, ly) {
			p, _, _, _ := parts(w, lx-w.Rect.X, ly-w.Rect.Y)
			return w.ID, p
		}
	}
	return "", PartNone
}

// ContentRect is where the window's content is on the screen (inside its
// border, offset by Bounds), for routing mouse events to the widget in it.
func (m Model) ContentRect(id string) (hittest.Rect, bool) {
	w, ok := m.Window(id)
	if !ok {
		return hittest.Rect{}, false
	}
	return hittest.Rect{X: m.Bounds.X + w.Rect.X + 1, Y: m.Bounds.Y + w.Rect.Y + 1, W: max(w.Rect.W-2, 0), H: max(w.Rect.H-2, 0)}, true
}

// Dragging reports whether a window is being moved or resized with the mouse.
func (m Model) Dragging() bool { return m.drag.on }

// Update obeys the KeyMap and, with Mouse on, the mouse. It returns a Cmd
// delivering ChangedMsg, ClosedMsg or FocusedMsg when that happened. Any other
// Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	switch ev := msg.(type) {
	case tui.MouseEvent:
		return m.updateMouse(ev)
	case tui.Key:
		return m.updateKey(ev)
	}
	return m, nil
}

func (m Model) updateKey(k tui.Key) (Model, tui.Cmd) {
	km := m.keys()
	if len(m.Windows) == 0 {
		return m, nil
	}
	step := max(m.Step, 1)
	top := len(m.Windows) - 1
	w := m.Windows[top]
	resize := func(dx, dy, dw, dh int) (Model, tui.Cmd) {
		if w.Fixed {
			return m, nil
		}
		r := w.Rect
		r.X, r.Y, r.W, r.H = r.X+dx*step, r.Y+dy*step, r.W+dw*step, r.H+dh*step
		return m.setRect(top, r)
	}
	switch {
	case keymap.Matches(k, km.Next) && len(m.Windows) > 1:
		m.own()
		m.Windows = append([]Window{m.Windows[top]}, m.Windows[:top]...)
		return m, focused(m.Focused())
	case keymap.Matches(k, km.Prev) && len(m.Windows) > 1:
		m.own()
		m.Windows = append(slices.Clone(m.Windows[1:]), m.Windows[0])
		return m, focused(m.Focused())
	case keymap.Matches(k, km.Close) && !w.NoClose:
		m.Remove(w.ID)
		return m, closed(w.ID)
	case keymap.Matches(k, km.Maximize) && !w.Fixed:
		return m.toggleMaximize(top)
	case keymap.Matches(k, km.MoveLeft):
		return resize(-1, 0, 0, 0)
	case keymap.Matches(k, km.MoveRight):
		return resize(1, 0, 0, 0)
	case keymap.Matches(k, km.MoveUp):
		return resize(0, -1, 0, 0)
	case keymap.Matches(k, km.MoveDown):
		return resize(0, 1, 0, 0)
	case keymap.Matches(k, km.ShrinkW):
		return resize(0, 0, -1, 0)
	case keymap.Matches(k, km.GrowW):
		return resize(0, 0, 1, 0)
	case keymap.Matches(k, km.ShrinkH):
		return resize(0, 0, 0, -1)
	case keymap.Matches(k, km.GrowH):
		return resize(0, 0, 0, 1)
	}
	return m, nil
}

// setRect puts window i at r and reports the change.
func (m Model) setRect(i int, r hittest.Rect) (Model, tui.Cmd) {
	m.own()
	old := m.Windows[i].Rect
	m.Windows[i].Rect = m.clamp(m.Windows[i], r)
	m.Windows[i].maximized = false
	if m.Windows[i].Rect == old {
		return m, nil
	}
	return m, changed(m.Windows[i])
}

func (m Model) toggleMaximize(i int) (Model, tui.Cmd) {
	m.own()
	w := &m.Windows[i]
	if w.maximized {
		w.Rect, w.maximized = m.clamp(*w, w.restore), false
	} else if dw, dh, ok := m.size(); ok {
		w.restore, w.Rect, w.maximized = w.Rect, hittest.Rect{W: dw, H: dh}, true
	}
	return m, changed(*w)
}

func changed(w Window) tui.Cmd {
	msg := ChangedMsg{ID: w.ID, Rect: w.Rect}
	return func() tui.Msg { return msg }
}
func closed(id string) tui.Cmd  { return func() tui.Msg { return ClosedMsg{ID: id} } }
func focused(id string) tui.Cmd { return func() tui.Msg { return FocusedMsg{ID: id} } }

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || m.Bounds.Empty() {
		return m, nil
	}
	lx, ly := m.Bounds.Local(ev.X, ev.Y)
	switch {
	case m.drag.on && ev.Action == tui.MouseActionMotion:
		return m.dragTo(lx, ly), nil
	case m.drag.on && ev.Action == tui.MouseActionRelease:
		d := m.drag
		m.drag = drag{}
		if i := m.index(d.id); i >= 0 && m.Windows[i].Rect != d.start {
			return m, changed(m.Windows[i])
		}
	case ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft:
		return m.press(lx, ly)
	}
	return m, nil
}

// press raises the window under the cell and starts a drag or closes it.
func (m Model) press(lx, ly int) (Model, tui.Cmd) {
	id, _ := m.Hit(lx+m.Bounds.X, ly+m.Bounds.Y)
	if id == "" {
		return m, nil
	}
	var cmd tui.Cmd
	if id != m.Focused() {
		m.Raise(id)
		cmd = focused(id)
	}
	w := m.Windows[len(m.Windows)-1]
	p, left, right, down := parts(w, lx-w.Rect.X, ly-w.Rect.Y)
	switch {
	case p == PartClose:
		m.Remove(id)
		return m, closed(id)
	case w.Fixed || p == PartBody:
	case p == PartTitle:
		m.drag = drag{on: true, id: id, move: true, offX: lx - w.Rect.X, offY: ly - w.Rect.Y, start: w.Rect}
	case p == PartEdge:
		m.drag = drag{on: true, id: id, left: left, right: right, down: down, start: w.Rect}
	}
	return m, cmd
}

// dragTo moves or resizes the dragged window so the pointer at (lx, ly) keeps
// its hold.
func (m Model) dragTo(lx, ly int) Model {
	i := m.index(m.drag.id)
	if i < 0 {
		m.drag = drag{}
		return m
	}
	d, w := m.drag, m.Windows[i]
	r := d.start
	if d.move {
		r.X, r.Y = lx-d.offX, ly-d.offY
	} else {
		minW, minH := max(w.MinW, defaultMinW), max(w.MinH, defaultMinH)
		if d.right {
			r.W = lx - r.X + 1
		}
		if d.down {
			r.H = ly - r.Y + 1
		}
		if d.left {
			edge := d.start.X + d.start.W
			r.X = min(lx, edge-minW)
			r.W = edge - r.X
		}
		r.W, r.H = max(r.W, minW), max(r.H, minH)
	}
	m.own()
	m.Windows[i].Rect = m.clamp(w, r)
	m.Windows[i].maximized = false
	return m
}

// View renders the desktop at Bounds' size.
func (m Model) View() string {
	return m.LayoutNode().Render(layout.Size{W: m.Bounds.W, H: m.Bounds.H})
}

// Linearize describes the desktop as plain text for accessible output (see
// tui.Linearizer): each window's title, whether it has focus, and its own
// linear text when its content has one.
func (m Model) Linearize() string {
	var b strings.Builder
	b.WriteString("Windows, " + strconv.Itoa(len(m.Windows)))
	for i := len(m.Windows) - 1; i >= 0; i-- {
		w := m.Windows[i]
		b.WriteString("\nWindow \"" + ansi.Sanitize(w.Title) + "\"")
		if i == len(m.Windows)-1 {
			b.WriteString(" (focused)")
		}
		if l, ok := w.Content.(tui.Linearizer); ok {
			b.WriteString("\n" + l.Linearize())
		}
	}
	return b.String()
}

// Compile-time proofs that Model satisfies the widget contracts.
var (
	_ tui.Component[Model] = Model{}
	_ tui.Linearizer       = Model{}
)
