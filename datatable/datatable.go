// Package datatable is widgets.Table plus row navigation — InkUI's
// "DataTable": cursor-based row highlighting and a SelectedMsg on top of
// widgets.Table, a column sort, and an optional filter line.
//
// The sort key orders the rows by the current column (numbers as numbers),
// ascending then descending; SortBy and ClearSort do the same from code. With
// Filter on, "/" opens a filter line: typed text keeps the rows whose cells
// fuzzy-match it, Enter closes the line and keeps the filter, Esc clears it.
// Rows is what is shown; SetRows replaces the rows given in, and SourceRow
// maps a shown row back to its place in them. Assign Rows directly only when
// no sort or filter is active.
package datatable

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Model is a bordered table with keyboard-navigable row selection:
// Up/Down move the cursor, Enter confirms the row under it.
type Model struct {
	Headers []string
	Rows    [][]string
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Raw, when true, draws Headers and Rows unchanged. By default every
	// header and cell is sanitised (ansi.Sanitize): escape sequences and
	// control characters are removed, so untrusted cell text cannot move the
	// cursor, clear the screen or write the clipboard.
	Raw bool

	// Height is the number of data rows shown at once (header and divider
	// not counted). Zero means "derive it": once the Model has seen a
	// tui.ResizeMsg it shows ResizeMsg.Height minus the two header/divider
	// lines (at least one row) and keeps the cursor visible; before any
	// ResizeMsg, zero renders every row. Column widths are computed from
	// the visible window only, so View costs O(Height), not O(len(Rows)).
	Height int

	// Width is the number of columns available to the table. Zero means
	// "no limit": every column is drawn. When the columns are wider than
	// Width, only a horizontal window of them is drawn and KeyMap.Left/Right
	// move the current column, scrolling the window to keep it visible.
	Width int

	// KeyMap holds the keys Update reacts to; New fills it with
	// DefaultKeyMap. A Model built as a struct literal with no KeyMap set
	// uses DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent inside Bounds:
	// the wheel scrolls by WheelStep rows and a left click selects the row
	// under the pointer. The zero value ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the table (its
	// top-left cell is the header line). The app sets it.
	Bounds hittest.Rect
	// Name is the layout.Named node the table is placed in; WithLayout
	// reads Bounds from that node's rectangle so the app sets no Bounds.
	Name string
	// WheelStep is the rows scrolled per wheel notch; zero means 3.
	WheelStep int

	// Filter, when true, lets KeyMap.Filter ("/") open a filter line above the
	// table: see the package comment. Off (the default), no key opens it and
	// the table is as before.
	Filter bool

	// ColumnWidths fixes column widths for LayoutNode: entry i, when positive,
	// is the width of column i; zero, a negative entry or a missing entry
	// leaves that column at its natural (content) width. Fixed widths do not
	// depend on the rows, so they stay constant while scrolling and cost no
	// row scan. Cells wider than their column are clipped with an ellipsis.
	// View is unaffected.
	ColumnWidths []int

	// widths caches natural column widths; see widthCache. Set by New and
	// SetRows, shared by copies of the Model.
	widths *widthCache

	cursor int
	offset int
	// col is the current column (the one Sort sorts by) and colOffset the
	// first drawn column.
	col, colOffset int
	// sortCol is the column the rows are ordered by, valid when sorted.
	sortCol  int
	sortDesc bool
	sorted   bool
	// resizeH is the last ResizeMsg height, used when Height == 0.
	resizeH int

	// query and editing are the filter line. src holds the rows in the
	// caller's order while a sort or filter makes Rows a different slice, and
	// idx maps each row of Rows to its place in src; both are nil otherwise.
	query   string
	editing bool
	src     [][]string
	idx     []int
}

// chrome is the header line plus the divider line above the data rows.
const chrome = 2

// cursorGutter is the width of the "> " marker column View draws at the left.
const cursorGutter = 2

// height is the effective data-row window: Height if set, else derived from
// the last ResizeMsg, else 0 (all rows).
func (m Model) height() int {
	if m.Height > 0 {
		return m.Height
	}
	if m.resizeH > 0 {
		return max(m.resizeH-chrome-m.top(), 1)
	}
	return 0
}

// defaultPage is the PgUp/PgDn step when Height is zero.
const defaultPage = 10

// window returns the first visible row and the row count, with the cursor
// kept inside the window.
func (m Model) window() (start, n int) {
	h := m.height()
	if h <= 0 || h >= len(m.Rows) {
		return 0, len(m.Rows)
	}
	start = clamp(m.offset, 0, len(m.Rows)-h)
	if m.cursor < start {
		start = m.cursor
	} else if m.cursor >= start+h {
		start = m.cursor - h + 1
	}
	return start, h
}

func (m *Model) reveal() { m.offset, _ = m.window() }

// New builds a Model from headers and rows, with the cursor on row 0.
func New(headers []string, rows [][]string) Model {
	return Model{Headers: headers, Rows: rows, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), widths: &widthCache{}}
}

// SetRows replaces the rows and invalidates the cached natural column widths.
// Call it (rather than assigning Rows) after editing cells in place, which
// the cache cannot see. The cursor is clamped to the new rows.
//
// With a sort or filter active the new rows are sorted and filtered the same
// way, and the cursor keeps its index.
func (m *Model) SetRows(rows [][]string) {
	m.widths = &widthCache{}
	if m.src != nil || m.sorted || m.query != "" {
		keep := m.cursor
		m.src = rows
		m.rebuild(false)
		m.SetCursor(keep)
		return
	}
	m.Rows = rows
	m.SetCursor(m.cursor)
}

// KeyMap names the keys Update reacts to.
type KeyMap struct {
	Up, Down         keymap.Binding
	PageUp, PageDown keymap.Binding
	Top, Bottom      keymap.Binding
	// Select confirms the row under the cursor (SelectedMsg).
	Select keymap.Binding
	// Sort orders the rows by the current column, toggling ascending and
	// descending on repeat.
	Sort keymap.Binding
	// Left and Right move the current column, scrolling wide tables.
	Left, Right keymap.Binding
	// Filter opens the filter line, and ClearSort puts the rows back in their
	// given order. Filter works only when Model.Filter is true.
	Filter, ClearSort keymap.Binding
}

// DefaultKeyMap returns the default keys: arrows, PgUp/PgDn, Home/End,
// Enter, "s" to sort, Left/Right for columns.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:        keymap.NewBinding("up", "up"),
		Down:      keymap.NewBinding("down", "down"),
		PageUp:    keymap.NewBinding("page up", "pgup"),
		PageDown:  keymap.NewBinding("page down", "pgdown"),
		Top:       keymap.NewBinding("top", "home"),
		Bottom:    keymap.NewBinding("bottom", "end"),
		Select:    keymap.NewBinding("select", "enter"),
		Sort:      keymap.NewBinding("sort", "s"),
		Left:      keymap.NewBinding("previous column", "left"),
		Right:     keymap.NewBinding("next column", "right"),
		Filter:    keymap.NewBinding("filter", "/"),
		ClearSort: keymap.NewBinding("unsort", "S"),
	}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	k := m.KeyMap
	if len(k.Up.Keys)+len(k.Down.Keys)+len(k.PageUp.Keys)+len(k.PageDown.Keys)+len(k.Top.Keys)+
		len(k.Bottom.Keys)+len(k.Select.Keys)+len(k.Sort.Keys)+len(k.Left.Keys)+len(k.Right.Keys)+
		len(k.Filter.Keys)+len(k.ClearSort.Keys) == 0 {
		return DefaultKeyMap()
	}
	return k
}

// Bindings returns the active bindings, for help widgets.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	out := []keymap.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom, k.Select, k.Sort, k.Left, k.Right}
	if m.Filter {
		out = append(out, k.Filter)
	}
	return out
}

// Offset returns the index of the first visible row.
func (m Model) Offset() int {
	start, _ := m.window()
	return start
}

// Sorted reports the sorted column and direction; ok is false while the
// rows are in the caller's order.
func (m Model) Sorted() (col int, desc, ok bool) { return m.sortCol, m.sortDesc, m.sorted }

// sortBy is the sort key's action: order the rows by column c, ascending on
// first use, toggling direction when c is already the sort column.
func (m *Model) sortBy(c int) {
	m.SortBy(c, m.sorted && m.sortCol == c && !m.sortDesc)
}

// fitCols returns how many columns, starting at first, fit in Width (at
// least one; all of them when Width is zero).
func (m Model) fitCols(first int) int {
	n := len(m.Headers)
	if m.Width <= 0 || first >= n {
		return max(n-first, 0)
	}
	start, cnt := m.window()
	total, k := cursorGutter, 0
	for c := first; c < n; c++ {
		w := ansi.Width(m.Headers[c])
		for _, row := range m.Rows[start : start+cnt] {
			if c < len(row) {
				w = max(w, ansi.Width(row[c]))
			}
		}
		if k > 0 {
			w += 2
		}
		if k > 0 && total+w > m.Width {
			break
		}
		total += w
		k++
	}
	return k
}

// moveCol moves the current column by d and scrolls it into view.
func (m *Model) moveCol(d int) {
	if len(m.Headers) == 0 {
		return
	}
	m.col = clamp(m.col+d, 0, len(m.Headers)-1)
	if m.col < m.colOffset {
		m.colOffset = m.col
	}
	for m.colOffset < m.col && m.col >= m.colOffset+m.fitCols(m.colOffset) {
		m.colOffset++
	}
}

// WithLayout sets Bounds to the rectangle of the layout node named Name in
// the tree under root laid out at s (typically the last frame's tree and
// size). With no Name or no such node Bounds is left unchanged.
func (m Model) WithLayout(root layout.Node, s layout.Size) Model {
	if m.Name == "" {
		return m
	}
	if r, ok := layout.RectOf(root, s, m.Name); ok {
		m.Bounds = r
	}
	return m
}

// mouse handles a wheel or left-click inside Bounds.
func (m Model) mouse(ev tui.MouseEvent) Model {
	if !m.Mouse || ev.Action != tui.MouseActionPress || !m.Bounds.Contains(ev.X, ev.Y) {
		return m
	}
	step := m.WheelStep
	if step <= 0 {
		step = 3
	}
	start, n := m.window()
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		if n >= len(m.Rows) {
			return m
		}
		if ev.Button == tui.MouseButtonWheelUp {
			step = -step
		}
		start = clamp(start+step, 0, len(m.Rows)-n)
		m.offset = start
		m.cursor = clamp(m.cursor, start, start+n-1)
	case tui.MouseButtonLeft:
		_, ly := m.Bounds.Local(ev.X, ev.Y)
		if row := start + ly - chrome - m.top(); ly >= chrome+m.top() && row < start+n {
			m.cursor = row
			m.reveal()
		}
	}
	return m
}

// Cursor returns the index of the row currently under the cursor.
func (m Model) Cursor() int { return m.cursor }

// SetCursor moves the cursor to row i, clamped to a valid index (or 0
// with no rows).
func (m *Model) SetCursor(i int) {
	if len(m.Rows) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = clamp(i, 0, len(m.Rows)-1)
	m.reveal()
}

// SelectedMsg is delivered (via the Cmd Update returns) when the row
// under the cursor is confirmed with Enter.
type SelectedMsg struct {
	Row   int
	Cells []string

	// SourceRow is the row's index among the rows given to New or SetRows;
	// Row is its index among those shown, which differs while the rows are
	// sorted or filtered.
	SourceRow int
}

// Update moves the cursor on Up/Down and confirms the row under it on
// Enter, returning a Cmd that delivers SelectedMsg. Non-Key Msgs and an
// empty Model are no-ops.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if rm, ok := msg.(tui.ResizeMsg); ok {
		m.resizeH = rm.Height
		m.reveal()
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if len(m.Rows) == 0 {
			return m, nil
		}
		return m.mouse(ev), nil
	}
	if _, ok := msg.(tui.Key); !ok {
		return m, nil
	}
	k := m.keys()
	if m.Filter {
		// The filter line works with no rows too, so a query that matches
		// nothing can still be edited away.
		if m.editing {
			if m.editFilter(msg, k) {
				return m, nil
			}
		} else if keymap.Matches(msg, k.Filter) {
			m.editing = true
			return m, nil
		} else if key := msg.(tui.Key); key.Type == tui.KeyEsc && m.query != "" {
			m.query = ""
			m.rebuild(false)
			return m, nil
		}
	}
	if len(m.Rows) == 0 {
		return m, nil
	}
	page := m.height()
	if page <= 0 {
		page = defaultPage
	}
	switch {
	case keymap.Matches(msg, k.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keymap.Matches(msg, k.Down):
		if m.cursor < len(m.Rows)-1 {
			m.cursor++
		}
	case keymap.Matches(msg, k.PageUp):
		m.cursor = clamp(m.cursor-page, 0, len(m.Rows)-1)
	case keymap.Matches(msg, k.PageDown):
		m.cursor = clamp(m.cursor+page, 0, len(m.Rows)-1)
	case keymap.Matches(msg, k.Top):
		m.cursor = 0
	case keymap.Matches(msg, k.Bottom):
		m.cursor = len(m.Rows) - 1
	case keymap.Matches(msg, k.Select):
		row, cells := m.cursor, m.Rows[m.cursor]
		src := m.SourceRow(row)
		return m, func() tui.Msg { return SelectedMsg{Row: row, SourceRow: src, Cells: cells} }
	case keymap.Matches(msg, k.Sort):
		if m.col < len(m.Headers) {
			m.sortBy(m.col)
		}
	case keymap.Matches(msg, k.ClearSort):
		m.ClearSort()
	case keymap.Matches(msg, k.Left):
		m.moveCol(-1)
	case keymap.Matches(msg, k.Right):
		m.moveCol(1)
	}
	m.reveal()
	return m, nil
}

// View renders the table with the row under the cursor highlighted, via
// widgets.TableRows — the same column-width computation, divider, and cell
// padding widgets.Table uses, so the two never drift apart.
func (m Model) View() string {
	headerStyle := ansi.NewStyle().Bold().Foreground(m.themed().Primary)
	dividerStyle := ansi.NewStyle().Foreground(m.themed().Muted)
	cursorStyle := m.themed().ResolvedStates().Selected.Bold()

	start, n := m.window()
	// Rows whose cells need no cleaning (the common case) are used as they are;
	// the row list and a row are copied only when something in them changes,
	// so a frame costs no per-row allocation.
	rows := m.Rows[start : start+n]
	owned := false
	own := func() {
		if !owned {
			rows = append([][]string(nil), rows...)
			owned = true
		}
	}
	for i, row := range rows {
		cleaned := false
		for j, c := range row {
			if cc := ansi.Clean(m.Raw, c); cc != c {
				if !cleaned {
					own()
					row = append([]string(nil), row...)
					rows[i] = row
					cleaned = true
				}
				row[j] = cc
			}
		}
	}
	headers := ansi.CleanAll(m.Raw, m.Headers)
	if m.sorted && m.sortCol < len(headers) {
		headers = append([]string(nil), headers...)
		arrow := " ^"
		if m.sortDesc {
			arrow = " v"
		}
		headers[m.sortCol] += arrow
	}
	if m.Width > 0 && len(headers) > 0 {
		first := clamp(m.colOffset, 0, len(headers)-1)
		last := min(first+max(m.fitCols(first), 1), len(headers))
		headers = headers[first:last]
		own()
		for i, r := range rows {
			rows[i] = r[min(first, len(r)):min(last, len(r))]
		}
	}
	table := widgets.TableRowsWith(headers, rows, headerStyle, dividerStyle, func(i int) ansi.Style {
		if i+start == m.cursor {
			return cursorStyle
		}
		return ansi.Style{}
	}, m.themed())
	out := withGutter(table, start, m.cursor)
	if m.top() > 0 {
		out = m.filterLine() + "\n" + out
	}
	return out
}

// withGutter prefixes every line of table with the marker column: "> " on the
// cursor row (table's data row i is row start+i), blanks elsewhere, so the
// cursor row shows without colour.
func withGutter(table string, start, cursor int) string {
	if table == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(table) + cursorGutter*(strings.Count(table, "\n")+1))
	for i, rest := 0, table; ; i++ {
		line, next, more := strings.Cut(rest, "\n")
		if i >= chrome && i-chrome+start == cursor {
			b.WriteString("> ")
		} else {
			b.WriteString("  ")
		}
		b.WriteString(line)
		if !more {
			return b.String()
		}
		b.WriteByte('\n')
		rest = next
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

// Linearize renders the table as plain text for accessible output (see
// tui.Linearizer): one self-contained line per row pairing each header
// with its cell, e.g. "Row 2 of 10: Name: x, Status: y", with ", selected"
// appended on the cursor row. No padding, dividers or styling.
func (m Model) Linearize() string {
	if len(m.Rows) == 0 {
		return "No rows"
	}
	lines := make([]string, len(m.Rows))
	for i, row := range m.Rows {
		cells := make([]string, len(row))
		for j, cell := range row {
			header := "Column " + strconv.Itoa(j+1)
			if j < len(m.Headers) {
				header = ansi.Clean(m.Raw, m.Headers[j])
			}
			cells[j] = header + ": " + ansi.Clean(m.Raw, cell)
		}
		line := "Row " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Rows)) + ": " + strings.Join(cells, ", ")
		if i == m.cursor {
			line += ", selected"
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}
