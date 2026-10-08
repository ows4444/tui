// Package otpinput is a field for a short code typed one character to a
// cell, such as a one-time password: each character fills a cell and moves
// to the next, Backspace steps back, and a paste fills as many cells as it
// has characters.
//
// Stability: experimental. Its API may change in any minor release.
package otpinput

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one code field. The zero value is not ready to use; build one
// with New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one field's change from
	// another's.
	ID string
	// Length is the number of cells. A Length under 1 is read as 6.
	Length int
	// Group, when above 0, draws a dash between every Group cells:
	// "[1][2][3] - [4][5][6]" for a Length of 6 and a Group of 3.
	Group int
	// Accept decides which characters a cell takes. Nil takes the digits 0
	// to 9. Use unicode.IsLetter, or a function of your own, for other
	// codes.
	Accept func(rune) bool
	// Upper turns a typed letter into its capital.
	Upper bool
	// Mask draws a filled cell with the theme's mask glyph, not its
	// character.
	Mask bool
	// Disabled stops the code being changed; it is drawn faint.
	Disabled bool
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// a cell moves the cursor to it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the field.
	Bounds hittest.Rect

	cells   []rune // 0 for an empty cell
	cursor  int
	focused bool
}

// New returns a field of length cells that takes digits.
func New(length int) Model {
	return Model{Length: length, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model. A character the field
// accepts always fills the cell under the cursor and is not in the KeyMap.
type KeyMap struct {
	Back  keymap.Binding // empty the cell under the cursor, or the one before it
	Prev  keymap.Binding // move to the cell before
	Next  keymap.Binding // move to the cell after
	First keymap.Binding // move to the first cell
	Last  keymap.Binding // move to the last cell
	Clear keymap.Binding // empty every cell
}

// DefaultKeyMap returns Backspace, Left and Right, Home and End, and Ctrl+U.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Back:  keymap.NewBinding("delete", "backspace"),
		Prev:  keymap.NewBinding("previous", "left"),
		Next:  keymap.NewBinding("next", "right"),
		First: keymap.NewBinding("first", "home"),
		Last:  keymap.NewBinding("last", "end"),
		Clear: keymap.NewBinding("clear", "ctrl+u"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Back.Keys)+len(km.Prev.Keys)+len(km.Next.Keys)+len(km.First.Keys)+len(km.Last.Keys)+len(km.Clear.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the field honours, with descriptions, for
// help text. A disabled field has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	km := m.keys()
	return []keymap.Binding{km.Back, km.Prev, km.Next, km.First, km.Last, km.Clear}
}

// ChangedMsg is delivered (via the Cmd Update returns) when the code
// changes.
type ChangedMsg struct {
	// ID is the field's ID.
	ID string
	// Value is the code so far: the filled cells in order, without the
	// empty ones.
	Value string
	// Complete reports whether every cell is filled.
	Complete bool
}

func (m Model) length() int {
	if m.Length < 1 {
		return 6
	}
	return m.Length
}

// grid returns the cells as a slice of exactly Length, whatever Length was
// when they were last written. It is a copy: a Model is a value, and a copy
// of it must not change through a slice the two share.
func (m Model) grid() []rune {
	g := make([]rune, m.length())
	copy(g, m.cells)
	return g
}

func (m Model) accepts(r rune) bool {
	if m.Accept != nil {
		return m.Accept(r)
	}
	return r >= '0' && r <= '9'
}

// Value returns the code so far: the filled cells in order, without the
// empty ones.
func (m Model) Value() string {
	var b strings.Builder
	for _, r := range m.grid() {
		if r != 0 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Complete reports whether every cell is filled.
func (m Model) Complete() bool {
	for _, r := range m.grid() {
		if r == 0 {
			return false
		}
	}
	return true
}

// SetValue empties the field and fills it from the start with the
// characters of s that it accepts, as a paste does, without a key press: no
// ChangedMsg is delivered. The cursor is left after the last one filled.
func (m *Model) SetValue(s string) {
	m.cells, m.cursor = nil, 0
	*m = m.fill(s)
}

// Cursor returns the index of the cell the cursor is on.
func (m Model) Cursor() int { return min(max(m.cursor, 0), m.length()-1) }

// Focus gives the field keyboard focus. It returns no Cmd; the result is
// there so a field can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the field has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// fill writes the accepted characters of s into the cells from the cursor
// on, one to a cell, and stops at the last cell. The cursor ends on the cell
// after the last one written, or on the last cell.
func (m Model) fill(s string) Model {
	g := m.grid()
	at := m.Cursor()
	for _, r := range s {
		if m.Upper {
			r = unicode.ToUpper(r)
		}
		if !m.accepts(r) {
			continue
		}
		if at >= len(g) {
			break
		}
		g[at] = r
		at++
	}
	m.cells, m.cursor = g, min(at, len(g)-1)
	return m
}

// report returns m and, when its code differs from before's, the Cmd that
// says so.
func (m Model) report(before string, wasComplete bool) (Model, tui.Cmd) {
	if m.Value() == before && m.Complete() == wasComplete {
		return m, nil
	}
	msg := ChangedMsg{ID: m.ID, Value: m.Value(), Complete: m.Complete()}
	return m, func() tui.Msg { return msg }
}

// Update fills the cell under the cursor with a typed character the field
// accepts and moves to the next; a pasted text fills as many cells as it has
// accepted characters. Backspace empties the cell under the cursor, or the
// one before it when that one is already empty. Left, Right, Home and End
// move the cursor, and Ctrl+U empties the field. With Mouse on, a left press
// on a cell moves the cursor to it. Only a field that has focus acts on
// keys, and a disabled one acts on nothing.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.Disabled {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft && m.Bounds.Contains(ev.X, ev.Y) {
			if i := m.cellAt(ev.X - m.Bounds.X); i >= 0 {
				m.cursor = i
			}
		}
		return m, nil
	}
	if !m.focused {
		return m, nil
	}
	before, was := m.Value(), m.Complete()
	if p, ok := msg.(tui.PasteEvent); ok {
		return m.fill(p.Text).report(before, was)
	}
	km := m.keys()
	last := m.length() - 1
	switch {
	case keymap.Matches(msg, km.Back):
		g := m.grid()
		at := m.Cursor()
		if g[at] == 0 && at > 0 {
			at--
		}
		g[at] = 0
		m.cells, m.cursor = g, at
		return m.report(before, was)
	case keymap.Matches(msg, km.Prev):
		m.cursor = max(m.Cursor()-1, 0)
	case keymap.Matches(msg, km.Next):
		m.cursor = min(m.Cursor()+1, last)
	case keymap.Matches(msg, km.First):
		m.cursor = 0
	case keymap.Matches(msg, km.Last):
		m.cursor = last
	case keymap.Matches(msg, km.Clear):
		m.cells, m.cursor = nil, 0
		return m.report(before, was)
	default:
		if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes && k.Text != "" {
			return m.fill(k.Text).report(before, was)
		}
	}
	return m, nil
}

// separator is what stands between two groups of cells.
func (m Model) separator() string { return " " + m.themed().GlyphSet().Dash + " " }

// grouped reports whether a separator stands before cell i.
func (m Model) grouped(i int) bool { return m.Group > 0 && i > 0 && i%m.Group == 0 }

// cellAt returns the cell drawn at local column x, or -1 for a separator or
// a column past the last cell.
func (m Model) cellAt(x int) int {
	left := 0
	sep := ansi.Width(m.separator())
	for i := 0; i < m.length(); i++ {
		if m.grouped(i) {
			left += sep
		}
		if x >= left && x < left+3 {
			return i
		}
		left += 3
	}
	return -1
}

// Width is the number of cells of the screen View takes: three for each
// cell of the code, and three for each separator.
func (m Model) Width() int {
	w := 3 * m.length()
	if m.Group > 0 {
		w += ansi.Width(m.separator()) * ((m.length() - 1) / m.Group)
	}
	return w
}

// View renders each cell as its character in square brackets, "[4]", and an
// empty one as "[ ]", so what has been typed reads without colour. The cell
// under the cursor of a field that has focus is drawn in angle brackets,
// "<4>" or "< >", as the other controls show focus. A disabled field is
// faint.
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	g := t.GlyphSet()
	filled := ansi.NewStyle().Foreground(t.Text)
	empty := ansi.NewStyle().Foreground(t.Muted)
	var b strings.Builder
	for i, r := range m.grid() {
		if m.grouped(i) {
			b.WriteString(empty.Render(m.separator()))
		}
		ch, style := " ", empty
		if r != 0 {
			ch, style = string(r), filled
			if m.Mask {
				ch = g.Mask
			}
		}
		open, shut := "[", "]"
		switch {
		case m.Disabled:
			style = st.Disabled.Faint()
		case m.focused && i == m.Cursor():
			open, shut = "<", ">"
			style = st.Focus.Bold()
		}
		b.WriteString(style.Render(open + ansi.Sanitize(ch) + shut))
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the field as plain text for accessible output (see
// tui.Linearizer): "code field, 3 of 6 entered", then the characters unless
// Mask is set, and ", unavailable" when disabled.
func (m Model) Linearize() string {
	v := m.Value()
	out := "code field, " + strconv.Itoa(len([]rune(v))) + " of " + strconv.Itoa(m.length()) + " entered"
	if v != "" && !m.Mask {
		out += ": " + ansi.Sanitize(v)
	}
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
