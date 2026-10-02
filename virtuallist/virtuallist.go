// Package virtuallist renders a scrollable window onto a large, uniform-height
// list without ever materializing off-screen rows. Unlike viewport.Model,
// which requires all content to be pre-rendered via SetContent, Model calls
// RenderItem only for the indices currently in view — so a list of 10,000+ items costs only as much
// rendering work as fits on screen, regardless of ItemCount.
package virtuallist

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// defaultOverscan is Overscan's default. Overscan no longer changes what View
// returns (see View); it is kept so existing callers still compile.
const defaultOverscan = 2

// Model is a fixed-height scrollable window onto ItemCount rows of content,
// each ItemHeight tall (informational only; the number of rows actually
// occupied by an item's rendering is up to RenderItem — Height and offset
// arithmetic here treat Height as a row count and every item as occupying
// one row of that count, mirroring viewport's line-based model). RenderItem
// is called on demand for exactly the indices needed to fill the window.
type Model struct {
	ItemCount  int
	ItemHeight int
	Height     int
	Overscan   int // accepted for compatibility; View renders exactly Height rows

	RenderItem func(i int) string

	Theme theme.Theme

	// KeyMap holds the keys Update reacts to. New sets DefaultKeyMap; a
	// Model built as a struct literal with a zero KeyMap uses DefaultKeyMap.
	KeyMap KeyMap

	// Mouse turns on mouse handling in Update (off by default). Bounds is the
	// screen rectangle where the app draws the list, set by the app; events
	// outside it are ignored. WheelStep is the rows one wheel notch scrolls
	// (0 means 3). A left press on a row moves the cursor to that item.
	Mouse     bool
	Bounds    hittest.Rect
	WheelStep int

	offset int
	cursor int
}

// KeyMap is the set of keys a list reacts to. Up and Down scroll the list by
// one item and carry the cursor with them; PageUp, PageDown, Top and Bottom
// jump the cursor and the view follows it.
type KeyMap struct {
	Up       keymap.Binding
	Down     keymap.Binding
	PageUp   keymap.Binding
	PageDown keymap.Binding
	Top      keymap.Binding
	Bottom   keymap.Binding
}

// DefaultKeyMap returns the default keys: up/down, pgup/pgdown, home/end.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       keymap.NewBinding("up", "up"),
		Down:     keymap.NewBinding("down", "down"),
		PageUp:   keymap.NewBinding("page up", "pgup"),
		PageDown: keymap.NewBinding("page down", "pgdown"),
		Top:      keymap.NewBinding("top", "home"),
		Bottom:   keymap.NewBinding("bottom", "end"),
	}
}

func (k KeyMap) isZero() bool {
	for _, b := range k.all() {
		if len(b.Keys) > 0 {
			return false
		}
	}
	return true
}

func (k KeyMap) all() []keymap.Binding {
	return []keymap.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom}
}

// keys is the KeyMap in force: KeyMap, or the defaults when it is unset.
func (m Model) keys() KeyMap {
	if m.KeyMap.isZero() {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the active key bindings, with descriptions, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().all() }

// New returns a Model that renders itemCount items in a window height rows
// tall, calling renderItem on demand. ItemHeight defaults to 1, Overscan to
// 2, and Theme to theme.DarkTheme(); set the fields on the returned Model to change
// them.
func New(itemCount, height int, renderItem func(i int) string) Model {
	return Model{
		ItemCount:  itemCount,
		ItemHeight: 1,
		Height:     height,
		Overscan:   defaultOverscan,
		RenderItem: renderItem,
		Theme:      theme.DarkTheme(),
		KeyMap:     DefaultKeyMap(),
	}
}

// visibleCount is how many item rows fit in the window at once.
func (m Model) visibleCount() int {
	if m.Height < 0 {
		return 0
	}
	return m.Height
}

// maxOffset is the largest offset that still leaves the window full of
// content: [0, ItemCount-visibleCount], never negative.
func (m Model) maxOffset() int {
	max := m.ItemCount - m.visibleCount()
	if max < 0 {
		return 0
	}
	return max
}

func (m *Model) setOffset(o int) { m.offset = clamp(o, 0, m.maxOffset()) }

// Offset reports the current scroll offset (index of the first item that
// would be visible with no overscan).
func (m Model) Offset() int { return m.offset }

// LineUp scrolls up n items, clamped to the top.
func (m *Model) LineUp(n int) { m.setOffset(m.offset - n) }

// LineDown scrolls down n items, clamped so the window never scrolls past
// the last item.
func (m *Model) LineDown(n int) { m.setOffset(m.offset + n) }

// Cursor reports the index of the cursor item, always in [0, ItemCount-1]
// (0 for an empty list).
func (m Model) Cursor() int { return clamp(m.cursor, 0, max(m.ItemCount-1, 0)) }

// SetCursor moves the cursor to item i (clamped) and scrolls the view just
// far enough to show it.
func (m *Model) SetCursor(i int) {
	m.cursor = clamp(i, 0, max(m.ItemCount-1, 0))
	m.follow()
}

// follow scrolls the view the minimum needed to show the cursor.
func (m *Model) follow() {
	m.cursor = m.Cursor()
	h := m.visibleCount()
	if m.cursor < m.offset {
		m.setOffset(m.cursor)
	} else if h > 0 && m.cursor >= m.offset+h {
		m.setOffset(m.cursor - h + 1)
	}
}

// pin moves the cursor into the visible window after a pure scroll.
func (m *Model) pin() {
	h := m.visibleCount()
	if h <= 0 {
		return
	}
	m.cursor = clamp(m.Cursor(), m.offset, m.offset+h-1)
	m.cursor = m.Cursor()
}

func (m Model) wheelStep() int {
	if m.WheelStep > 0 {
		return m.WheelStep
	}
	return 3
}

// Update handles the KeyMap keys and, when Mouse is on, mouse events inside
// Bounds, mirroring viewport.Model's method-based Update convention. Up and
// Down scroll one item and move the cursor one item; PageUp, PageDown, Top
// and Bottom move the cursor by a window or to either end and the view
// follows it. The wheel scrolls WheelStep items and keeps the cursor in the
// window; a left press on a row moves the cursor there.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.mouse(ev), nil
	}
	km := m.keys()
	page := max(m.visibleCount(), 1)
	switch {
	case keymap.Matches(msg, km.Up):
		m.LineUp(1)
		m.cursor = m.Cursor() - 1
		m.follow()
	case keymap.Matches(msg, km.Down):
		m.LineDown(1)
		m.cursor = m.Cursor() + 1
		m.follow()
	case keymap.Matches(msg, km.PageUp):
		m.SetCursor(m.Cursor() - page)
	case keymap.Matches(msg, km.PageDown):
		m.SetCursor(m.Cursor() + page)
	case keymap.Matches(msg, km.Top):
		m.SetCursor(0)
	case keymap.Matches(msg, km.Bottom):
		m.SetCursor(m.ItemCount - 1)
	}
	return m, nil
}

func (m Model) mouse(ev tui.MouseEvent) Model {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.LineUp(m.wheelStep())
		m.pin()
	case tui.MouseButtonWheelDown:
		m.LineDown(m.wheelStep())
		m.pin()
	case tui.MouseButtonLeft:
		_, ly := m.Bounds.Local(ev.X, ev.Y)
		if ly < m.visibleCount() && m.offset+ly < m.ItemCount {
			m.SetCursor(m.offset + ly)
		}
	}
	return m
}

// View renders the visible window: exactly Height rows, starting at the
// current offset (fewer only when ItemCount runs out). RenderItem is called
// for those indices and no others. Overscan is kept for compatibility but no
// longer widens the output: the extra rows were returned to the caller, so a
// list with Height 10 and Overscan 2 showed 14 rows and overflowed its box.
// If ItemCount <= 0 or Height <= 0, View returns "" and RenderItem is never
// called.
func (m Model) View() string {
	if m.ItemCount <= 0 || m.Height <= 0 || m.RenderItem == nil {
		return ""
	}

	start := clamp(m.offset, 0, m.ItemCount)
	end := start + m.visibleCount()
	if end > m.ItemCount {
		end = m.ItemCount
	}

	rows := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, m.RenderItem(i))
	}
	return strings.Join(rows, "\n")
}

// DrawCells draws the visible window into the region r of buf, as View would
// show it (it implements tui.CellDrawer). RenderItem is called for the visible
// items only, and each item's styled text is written straight into the cells of
// its row: no window string is built, joined or split. An item that renders
// several lines takes several rows; rows past r are clipped. An item the cell
// grid cannot represent (see cellbuf.ErrUnsupported) is drawn as plain text with
// its escape sequences removed. Nothing is drawn when ItemCount <= 0, Height <=
// 0 or RenderItem is nil, as View returns "" then.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	if m.ItemCount <= 0 || m.Height <= 0 || m.RenderItem == nil {
		return
	}
	sub := buf.Sub(r)
	start := clamp(m.offset, 0, m.ItemCount)
	end := min(start+m.visibleCount(), m.ItemCount)
	y := 0
	for i := start; i < end && y < sub.Height(); i++ {
		for rest, more := m.RenderItem(i), true; more && y < sub.Height(); y++ {
			var line string
			line, rest, more = strings.Cut(rest, "\n")
			line = strings.TrimSuffix(line, "\r")
			if line == "" {
				continue
			}
			if _, err := sub.SetStyled(0, y, line); err != nil {
				sub.SetString(0, y, line, 0)
			}
		}
	}
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

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Compile-time proof that Model draws into a cell grid; tui.CellDrawer is
// structural, so no import of it is needed.
var _ interface {
	DrawCells(*cellbuf.Buffer, cellbuf.Rect)
} = Model{}
