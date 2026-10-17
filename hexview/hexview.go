// Package hexview shows bytes as a hex dump: an offset, the bytes in hex, in
// groups, and an ASCII gutter, with a cursor you move over the bytes, a
// selection you can copy as hex, and the mouse (click a byte, drag to select,
// wheel to scroll) when it is on.
//
// It holds the bytes it is given and does not copy them again per frame, so a
// large buffer costs one row of work per visible row.
//
// Stability: experimental. Its API may change in any minor release.
package hexview

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is a hex dump of Data with a cursor. Build one with New.
type Model struct {
	// Data is the bytes shown. Use SetData to replace it, so the cursor and
	// scroll position are kept inside it; assigning Data directly is fine for a
	// first value.
	Data []byte
	// BytesPerRow is how many bytes a row shows; 0 means 16.
	BytesPerRow int
	// GroupSize is how many bytes are drawn together before an extra space; 0
	// means 8 (the classic hexdump look) and a value of BytesPerRow or more
	// draws no groups.
	GroupSize int
	// Height is the number of rows View shows; 0 shows every row. With a Height
	// the view scrolls, by the least distance, to keep the cursor in sight.
	Height int
	// HideOffset and HideASCII leave out the offset column and the ASCII
	// gutter.
	HideOffset, HideASCII bool
	Theme                 theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap; a
	// Model built as a struct literal with a zero KeyMap behaves as if it held
	// DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on a
	// byte (in the hex or the ASCII column) puts the cursor on it and starts a
	// selection, a drag extends it, and the wheel scrolls. Off (the default)
	// ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the view (its first
	// row is the first row of the dump). The app sets it each frame it moves.
	Bounds hittest.Rect

	cursor   int
	anchor   int // the other end of the selection, or -1 for none
	top      int // first visible row when Height > 0
	dragging bool
}

// New returns a hex view of data with the cursor on its first byte.
func New(data []byte) Model {
	return Model{Data: data, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), anchor: -1}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Left, Right      keymap.Binding // one byte back, one byte on
	Up, Down         keymap.Binding // one row up, one row down
	PageUp, PageDown keymap.Binding // a screenful (Height rows, or 16)
	LineStart        keymap.Binding // the first byte of the row
	LineEnd          keymap.Binding // the last byte of the row
	Top, Bottom      keymap.Binding // the first and the last byte
	// Select starts a selection at the cursor, or ends it when one is going;
	// while it is on, moving the cursor extends it.
	Select keymap.Binding
	// Copy returns the selected bytes as hex text in a tui.WriteClipboard Cmd.
	Copy keymap.Binding
}

// DefaultKeyMap returns arrows, PgUp/PgDn, Home/End, ctrl+Home/End, "v" to
// select and "y" to copy.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left:      keymap.NewBinding("back one byte", "left", "h"),
		Right:     keymap.NewBinding("on one byte", "right", "l"),
		Up:        keymap.NewBinding("up one row", "up", "k"),
		Down:      keymap.NewBinding("down one row", "down", "j"),
		PageUp:    keymap.NewBinding("page up", "pgup"),
		PageDown:  keymap.NewBinding("page down", "pgdown"),
		LineStart: keymap.NewBinding("start of row", "home"),
		LineEnd:   keymap.NewBinding("end of row", "end"),
		Top:       keymap.NewBinding("first byte", "ctrl+home", "g"),
		Bottom:    keymap.NewBinding("last byte", "ctrl+end", "G"),
		Select:    keymap.NewBinding("select", "v"),
		Copy:      keymap.NewBinding("copy as hex", "y"),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.Left, km.Right, km.Up, km.Down, km.PageUp, km.PageDown,
		km.LineStart, km.LineEnd, km.Top, km.Bottom, km.Select, km.Copy}
}

func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.list() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the actions the view honours, with descriptions, for help
// text.
func (m Model) Bindings() []keymap.Binding { return m.keys().list() }

func (m Model) perRow() int {
	if m.BytesPerRow < 1 {
		return 16
	}
	return m.BytesPerRow
}

func (m Model) group() int {
	if m.GroupSize < 1 {
		return 8
	}
	return m.GroupSize
}

// rows is the number of rows the data fills: at least one.
func (m Model) rows() int { return max((len(m.Data)+m.perRow()-1)/m.perRow(), 1) }

// SetData replaces the bytes, keeping the cursor on the same offset when it is
// still inside them (else on the last byte) and ending any selection.
func (m *Model) SetData(data []byte) {
	m.Data = data
	m.anchor = -1
	m.cursor = min(m.cursor, max(len(data)-1, 0))
	m.scrollToCursor()
}

// Cursor returns the offset of the byte under the cursor (0 when there is no
// data).
func (m Model) Cursor() int { return min(m.cursor, max(len(m.Data)-1, 0)) }

// SetCursor moves the cursor to offset i, held inside the data, and keeps it in
// sight. A selection in progress is extended to it.
func (m *Model) SetCursor(i int) {
	m.cursor = min(max(i, 0), max(len(m.Data)-1, 0))
	m.scrollToCursor()
}

// Selection returns the selected range as offsets from (first byte) and to
// (last byte, included), and whether a selection is going.
func (m Model) Selection() (from, to int, ok bool) {
	if m.anchor < 0 || len(m.Data) == 0 {
		return 0, 0, false
	}
	a, c := min(m.anchor, len(m.Data)-1), m.Cursor()
	return min(a, c), max(a, c), true
}

// SelectedBytes returns the selected bytes, or just the byte under the cursor
// when there is no selection, or nil when there is no data. The slice is a
// view of Data: do not modify it.
func (m Model) SelectedBytes() []byte {
	if len(m.Data) == 0 {
		return nil
	}
	if from, to, ok := m.Selection(); ok {
		return m.Data[from : to+1]
	}
	return m.Data[m.Cursor() : m.Cursor()+1]
}

// ClearSelection ends the selection.
func (m *Model) ClearSelection() { m.anchor = -1 }

// hexText is bytes as lower-case hex pairs separated by spaces.
func hexText(b []byte) string {
	const digits = "0123456789abcdef"
	var sb strings.Builder
	sb.Grow(len(b) * 3)
	for i, v := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteByte(digits[v>>4])
		sb.WriteByte(digits[v&15])
	}
	return sb.String()
}

// scrollToCursor moves the first visible row by the least distance that shows
// the cursor's row.
func (m *Model) scrollToCursor() {
	if m.Height <= 0 {
		m.top = 0
		return
	}
	row := m.Cursor() / m.perRow()
	switch {
	case row < m.top:
		m.top = row
	case row >= m.top+m.Height:
		m.top = row - m.Height + 1
	}
	m.top = min(max(m.top, 0), max(m.rows()-m.Height, 0))
}

// Update moves the cursor and selection on keys, and on the mouse when Mouse is
// on.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		return m.updateKey(msg)
	case tui.MouseEvent:
		return m.updateMouse(msg), nil
	}
	return m, nil
}

func (m Model) updateKey(k tui.Key) (Model, tui.Cmd) {
	km := m.keys()
	per, page := m.perRow(), m.pageRows()*m.perRow()
	c := m.Cursor()
	switch {
	case keymap.Matches(k, km.Left):
		m.SetCursor(c - 1)
	case keymap.Matches(k, km.Right):
		m.SetCursor(c + 1)
	case keymap.Matches(k, km.Up):
		m.SetCursor(c - per)
	case keymap.Matches(k, km.Down):
		m.SetCursor(c + per)
	case keymap.Matches(k, km.PageUp):
		m.SetCursor(c - page)
	case keymap.Matches(k, km.PageDown):
		m.SetCursor(c + page)
	case keymap.Matches(k, km.LineStart):
		m.SetCursor(c - c%per)
	case keymap.Matches(k, km.LineEnd):
		m.SetCursor(c - c%per + per - 1)
	case keymap.Matches(k, km.Top):
		m.SetCursor(0)
	case keymap.Matches(k, km.Bottom):
		m.SetCursor(len(m.Data) - 1)
	case keymap.Matches(k, km.Select):
		if m.anchor < 0 {
			m.anchor = c
		} else {
			m.anchor = -1
		}
	case keymap.Matches(k, km.Copy):
		if len(m.Data) > 0 {
			return m, tui.WriteClipboard(hexText(m.SelectedBytes()))
		}
	}
	return m, nil
}

func (m Model) pageRows() int {
	if m.Height > 0 {
		return m.Height
	}
	return 16
}

// ---- layout of one row ----

// offsetDigits is the width of the offset column: four hex digits, or eight
// once the data is longer than 64 KiB.
func (m Model) offsetDigits() int {
	if len(m.Data) > 0xFFFF {
		return 8
	}
	return 4
}

// hexCol is the column, from the start of the hex area, of the first digit of
// the byte in column c of a row.
func (m Model) hexCol(c int) int { return c*3 + c/m.group() }

// hexWidth is the width of the hex area, a row's bytes and their gaps.
func (m Model) hexWidth() int { return m.hexCol(m.perRow()-1) + 2 }

// hexStart and asciiStart are the columns where the hex area and the ASCII
// gutter's first character begin.
func (m Model) hexStart() int {
	if m.HideOffset {
		return 0
	}
	return m.offsetDigits() + 2
}

func (m Model) asciiStart() int { return m.hexStart() + m.hexWidth() + 3 }

// printable is the character drawn for a byte in the ASCII gutter.
func printable(b byte) byte {
	if b >= 0x20 && b < 0x7f {
		return b
	}
	return '.'
}

// View renders the dump: Height rows, or all of them when Height is 0.
func (m Model) View() string {
	st := m.styles()
	per := m.perRow()
	first, last := 0, m.rows()
	if m.Height > 0 {
		first = min(m.top, max(last-m.Height, 0))
		last = min(first+m.Height, last)
	}
	from, to, sel := m.Selection()
	cur := m.Cursor()
	hot := func(i int) (ansi.Style, bool) {
		switch {
		case len(m.Data) > 0 && i == cur:
			return st.cursor, true
		case sel && i >= from && i <= to:
			return st.selected, true
		}
		return ansi.Style{}, false
	}

	var rows []string
	for r := first; r < last; r++ {
		var b strings.Builder
		if !m.HideOffset {
			b.WriteString(st.offset.Render(pad(strconv.FormatInt(int64(r*per), 16), m.offsetDigits())))
			b.WriteString("  ")
		}
		for c := 0; c < per; c++ {
			i := r*per + c
			if c > 0 {
				b.WriteByte(' ')
				if c%m.group() == 0 {
					b.WriteByte(' ')
				}
			}
			if i >= len(m.Data) {
				b.WriteString("  ")
				continue
			}
			cell := hexText(m.Data[i : i+1])
			if h, ok := hot(i); ok {
				b.WriteString(h.Render(cell))
			} else if m.Data[i] == 0 {
				b.WriteString(st.zero.Render(cell))
			} else {
				b.WriteString(cell)
			}
		}
		if !m.HideASCII {
			b.WriteString("  |")
			for c := 0; c < per && r*per+c < len(m.Data); c++ {
				i := r*per + c
				ch := string(printable(m.Data[i]))
				if h, ok := hot(i); ok {
					b.WriteString(h.Render(ch))
				} else if ch == "." {
					b.WriteString(st.zero.Render(ch))
				} else {
					b.WriteString(ch)
				}
			}
			b.WriteString("|")
		}
		rows = append(rows, b.String())
	}
	return strings.Join(rows, "\n")
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat("0", width-len(s)) + s
}

type styleSet struct {
	offset, zero, cursor, selected ansi.Style
}

func (m Model) styles() styleSet {
	th := m.themed()
	sel := th.ResolvedStates().Selected
	return styleSet{
		offset:   ansi.NewStyle().Foreground(th.Muted),
		zero:     ansi.NewStyle().Faint(),
		cursor:   sel.Bold(),
		selected: sel,
	}
}
