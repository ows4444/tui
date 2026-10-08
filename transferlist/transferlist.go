// Package transferlist is two lists side by side with items that move
// between them: what is available on the left, what was chosen on the right.
// Up and Down move a cursor in one list, Left and Right change list, and
// Enter or Space moves the item under the cursor across. With the mouse on,
// a click on an item moves it.
//
// Stability: experimental. Its API may change in any minor release.
package transferlist

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Side names one of the two lists.
type Side int

// The two lists.
const (
	Left Side = iota
	Right
)

// other is the list across from s.
func (s Side) other() Side { return 1 - s }

// Model is one pair of lists. The zero value is two empty lists; build one
// with New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one pair's change from another's.
	ID string
	// Left and Right are the items of the two lists, in the order drawn. An
	// item that moves is added to the end of the other list. Update never
	// writes into the arrays behind them: a move builds new slices.
	Left, Right []string
	// LeftTitle and RightTitle head the two lists. Empty is read as
	// "Available" and "Chosen".
	LeftTitle, RightTitle string
	// Height is the number of item rows drawn. A list with more scrolls to
	// keep its cursor in view. Zero or less draws every item of the longer
	// list.
	Height int
	// ColumnWidth is the width of each list in cells; longer text is cut
	// and ends in an ellipsis. 1 is read as 2, the two end cells.
	// Zero or less fits each list to its title and its items, so a list's
	// width changes as items move.
	ColumnWidth int
	// Disabled draws the lists faint and makes Update ignore keys and the
	// pointer.
	Disabled bool
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// an item moves it across, and the wheel scrolls the list under it. Off
	// (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the lists.
	Bounds hittest.Rect

	focused bool
	side    Side
	cursor  [2]int
	offset  [2]int
}

// New returns a pair of lists with the given items on the left and none on
// the right.
func New(items ...string) Model {
	return Model{Left: items, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Up      keymap.Binding // move the cursor up
	Down    keymap.Binding // move the cursor down
	First   keymap.Binding // move the cursor to the first item
	Last    keymap.Binding // move the cursor to the last item
	ToLeft  keymap.Binding // put the cursor in the left list
	ToRight keymap.Binding // put the cursor in the right list
	Move    keymap.Binding // move the item under the cursor across
	MoveAll keymap.Binding // move every item of the cursor's list across
}

// DefaultKeyMap returns Up and Down, Home and End, Left and Right, Enter or
// Space to move one item, and Ctrl+A to move them all.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      keymap.NewBinding("up", "up"),
		Down:    keymap.NewBinding("down", "down"),
		First:   keymap.NewBinding("first", "home"),
		Last:    keymap.NewBinding("last", "end"),
		ToLeft:  keymap.NewBinding("left list", "left"),
		ToRight: keymap.NewBinding("right list", "right"),
		Move:    keymap.NewBinding("move", "enter", "space"),
		MoveAll: keymap.NewBinding("move all", "ctrl+a"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	n := len(km.Up.Keys) + len(km.Down.Keys) + len(km.First.Keys) + len(km.Last.Keys) +
		len(km.ToLeft.Keys) + len(km.ToRight.Keys) + len(km.Move.Keys) + len(km.MoveAll.Keys)
	if n == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the lists honour, with descriptions, for help
// text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Up, km.Down, km.First, km.Last, km.ToLeft, km.ToRight, km.Move, km.MoveAll}
}

// ChangedMsg is delivered (via the Cmd Update returns) when items move, by
// key or by pointer. The lists as they now stand are the Model's Left and
// Right.
type ChangedMsg struct {
	// ID is the pair's ID.
	ID string
	// Moved are the items that moved, in the order they were added.
	Moved []string
	// To is the list they moved to.
	To Side
}

func (m Model) items(s Side) []string {
	if s == Right {
		return m.Right
	}
	return m.Left
}

// Active returns the list the cursor is in. It is never an empty list while
// the other has items.
func (m Model) Active() Side {
	if len(m.items(m.side)) == 0 && len(m.items(m.side.other())) > 0 {
		return m.side.other()
	}
	return m.side
}

// Cursor returns the index of the item under the cursor in list s, or -1
// when that list is empty. Each list keeps its own.
func (m Model) Cursor(s Side) int {
	return min(max(m.cursor[s], 0), len(m.items(s))-1)
}

// SetCursor puts the cursor on item i of list s. An index out of range is
// ignored.
func (m *Model) SetCursor(s Side, i int) {
	if (s != Left && s != Right) || i < 0 || i >= len(m.items(s)) {
		return
	}
	m.side, m.cursor[s] = s, i
	m.offset[s] = m.top(s)
}

// Focus gives the lists keyboard focus. It returns no Cmd; the result is
// there so they can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the lists have keyboard focus.
func (m Model) Focused() bool { return m.focused }

// rows is the number of item rows drawn.
func (m Model) rows() int {
	if m.Height > 0 {
		return m.Height
	}
	return max(len(m.Left), len(m.Right), 1)
}

// top is the index of the first item of list s that is drawn: the stored
// offset, moved as little as it takes to show the cursor and to leave no
// blank rows under a list that could fill them.
func (m Model) top(s Side) int {
	h, c := m.rows(), m.Cursor(s)
	off := min(max(m.offset[s], c-h+1), max(c, 0))
	return min(max(off, 0), max(len(m.items(s))-h, 0))
}

// moveCursor steps the cursor of the active list by d items, stopping at
// either end.
func (m Model) moveCursor(d int) Model {
	s := m.Active()
	m.side = s
	m.cursor[s] = min(max(m.Cursor(s)+d, 0), max(len(m.items(s))-1, 0))
	m.offset[s] = m.top(s)
	return m
}

// transfer moves n items of list from, starting at item i, to the end of the
// other list.
func (m Model) transfer(from Side, i, n int) (Model, tui.Cmd) {
	src := m.items(from)
	moved := append([]string(nil), src[i:i+n]...)
	rest := append(append([]string(nil), src[:i]...), src[i+n:]...)
	dst := append(append([]string(nil), m.items(from.other())...), moved...)
	if from == Left {
		m.Left, m.Right = rest, dst
	} else {
		m.Left, m.Right = dst, rest
	}
	m.side = from
	m.cursor[from] = max(min(i, len(rest)-1), 0)
	m.side = m.Active()
	m.offset[Left], m.offset[Right] = m.top(Left), m.top(Right)
	msg := ChangedMsg{ID: m.ID, Moved: moved, To: from.other()}
	return m, func() tui.Msg { return msg }
}

// Update moves the cursor, changes list and moves items on the keys of the
// KeyMap when the lists have focus. With Mouse on, a left press on an item
// moves it across and the wheel scrolls the list under the pointer. Left or
// Right towards an empty list does nothing.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.Disabled {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.mouse(ev)
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	s := m.Active()
	n := len(m.items(s))
	switch {
	case keymap.Matches(msg, km.Up):
		return m.moveCursor(-1), nil
	case keymap.Matches(msg, km.Down):
		return m.moveCursor(1), nil
	case keymap.Matches(msg, km.First):
		return m.moveCursor(-n), nil
	case keymap.Matches(msg, km.Last):
		return m.moveCursor(n), nil
	case keymap.Matches(msg, km.ToLeft):
		if len(m.Left) > 0 {
			m.side = Left
		}
	case keymap.Matches(msg, km.ToRight):
		if len(m.Right) > 0 {
			m.side = Right
		}
	case keymap.Matches(msg, km.Move):
		if n > 0 {
			return m.transfer(s, m.Cursor(s), 1)
		}
	case keymap.Matches(msg, km.MoveAll):
		if n > 0 {
			return m.transfer(s, 0, n)
		}
	}
	return m, nil
}

func (m Model) mouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || ev.Action != tui.MouseActionPress || !m.Bounds.Contains(ev.X, ev.Y) {
		return m, nil
	}
	s, ok := m.sideAt(ev.X - m.Bounds.X)
	if !ok {
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		d := 1
		if ev.Button == tui.MouseButtonWheelUp {
			d = -1
		}
		m.offset[s] = min(max(m.top(s)+d, 0), max(len(m.items(s))-m.rows(), 0))
		// The cursor stays in view, so it follows a list scrolled past it.
		m.cursor[s] = min(max(m.Cursor(s), m.offset[s]), m.offset[s]+m.rows()-1)
	case tui.MouseButtonLeft:
		// Row 0 is the titles.
		if i := m.top(s) + ev.Y - m.Bounds.Y - 1; ev.Y > m.Bounds.Y && i < len(m.items(s)) {
			return m.transfer(s, i, 1)
		}
	}
	return m, nil
}

// sideAt returns the list drawn at local column x; ok is false on the rule
// between the two and outside both.
func (m Model) sideAt(x int) (s Side, ok bool) {
	l := m.width(Left)
	switch {
	case x >= 0 && x < l:
		return Left, true
	case x >= l+gap && x < l+gap+m.width(Right):
		return Right, true
	}
	return Left, false
}

// gap is the width of what stands between the lists: a rule with a blank
// cell either side.
const gap = 3

func (m Model) title(s Side) string {
	t, fallback := m.LeftTitle, "Available"
	if s == Right {
		t, fallback = m.RightTitle, "Chosen"
	}
	if t = ansi.Sanitize(t); t == "" {
		t = fallback
	}
	return t + " (" + strconv.Itoa(len(m.items(s))) + ")"
}

// width is the width of list s: its text with a cell either side.
func (m Model) width(s Side) int {
	if m.ColumnWidth > 0 {
		return max(m.ColumnWidth, 2)
	}
	w := ansi.Width(m.title(s))
	for _, it := range m.items(s) {
		w = max(w, ansi.Width(ansi.Sanitize(it)))
	}
	return w + 2
}

// Width is the number of cells View takes: the two lists and the rule
// between them.
func (m Model) Width() int { return m.width(Left) + gap + m.width(Right) }

// Rows is the number of rows View takes: the titles and the item rows.
func (m Model) Rows() int { return 1 + m.rows() }

// cell fits text into a list w cells wide, between the two given end cells.
// Text that is too long is cut and ends in the ellipsis.
func cell(open, text, shut, ellipsis string, w int) string {
	if room := max(w-2, 0); ansi.Width(text) > room {
		text = ansi.Truncate(text, max(room-1, 0))
		if room > 0 {
			text += ellipsis
		}
	}
	return open + text + shut + strings.Repeat(" ", max(w-2-ansi.Width(text), 0))
}

// View renders the two lists side by side under their titles, each title
// with its count, and a rule between them:
//
//	Available (2)  │  Chosen (1)
//	Apples         │  Dates
//	Cherries       │
//
// Every item has a blank cell either side. The item under the cursor of
// lists that have focus takes angle brackets in those two cells, "<Apples>",
// so focus reads without colour and does not change the width.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	item := ansi.NewStyle().Foreground(t.Text)
	head := ansi.NewStyle().Bold().Foreground(t.Primary)
	muted := ansi.NewStyle().Foreground(t.Muted)
	if m.Disabled {
		item, head = st.Disabled, st.Disabled
	}
	dots := t.GlyphSet().Ellipsis
	rule := " " + muted.Render(t.GlyphSet().RuleV) + " "
	active := m.Active()
	var b strings.Builder
	for row := range m.Rows() {
		if row > 0 {
			b.WriteByte('\n')
		}
		for _, s := range []Side{Left, Right} {
			if s == Right {
				b.WriteString(rule)
			}
			w := m.width(s)
			if row == 0 {
				b.WriteString(head.Render(cell(" ", m.title(s), " ", dots, w)))
				continue
			}
			its := m.items(s)
			i := m.top(s) + row - 1
			switch {
			case i >= len(its):
				b.WriteString(strings.Repeat(" ", w))
			case m.focused && !m.Disabled && s == active && i == m.Cursor(s):
				b.WriteString(st.Focus.Bold().Render(cell("<", ansi.Sanitize(its[i]), ">", dots, w)))
			default:
				b.WriteString(item.Render(cell(" ", ansi.Sanitize(its[i]), " ", dots, w)))
			}
		}
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the lists as plain text for accessible output (see
// tui.Linearizer): each title with its count, then its items one to a line,
// every item whether or not it is scrolled into view.
func (m Model) Linearize() string {
	var lines []string
	for _, s := range []Side{Left, Right} {
		lines = append(lines, m.title(s))
		for _, it := range m.items(s) {
			lines = append(lines, "  "+ansi.Sanitize(it))
		}
	}
	return strings.Join(lines, "\n")
}
