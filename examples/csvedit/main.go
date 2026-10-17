// Command csvedit edits a CSV file as a grid: move a cursor over the cells,
// change one, add and remove rows and columns, sort by a column, undo, and
// save. Numbers are right-aligned and sorted as numbers. With no file it opens
// a small sample and Save asks for a name.
//
// Mouse: click a cell to select it and click it again to edit, click a header
// to select it and sort by that column, scroll the wheel, click the toolbar.
// Keyboard: arrows or hjkl move, Enter or F2 edits (Enter saves the edit, Esc
// drops it), o and O add a row below or above, d deletes the row, + adds a
// column after this one, - deletes the column, s sorts (again to reverse),
// Ctrl+Z and Ctrl+Y undo and redo, c copies the cell, Ctrl+S saves, F1 shows
// help, q or Ctrl+Q quits (asking first if changes would be lost), Ctrl+C quits
// at once. The cursor can sit on a header: Enter renames the column.
//
//	go run ./examples/csvedit [file.csv]
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	minColW = 4
	maxColW = 24
	chrome  = 5 // title, toolbar, header, status, hints
	undoMax = 100
)

type mode int

const (
	modeGrid mode = iota
	modeEdit
	modeNewCol
	modeSaveAs
	modeDiscard
)

type model struct {
	st    store
	path  string
	cur   sheet
	saved string // the CSV as last saved or opened

	undo, redo []sheet
	row, col   int // row -1 is the header
	top, left  int // first data row and first column shown

	sortCol  int // column the rows were last sorted by, or -1
	sortDesc bool

	mode  mode
	in    textinput.Model
	msg   string
	help  bool
	hover string

	width, height int
}

const sample = `city,country,population,area_km2
Tokyo,Japan,37400000,2194
Delhi,India,31000000,1484
Shanghai,China,27100000,6341
São Paulo,Brazil,22000000,1521
Mexico City,Mexico,21800000,1485
Cairo,Egypt,20900000,3085
Mumbai,India,20400000,603
Beijing,China,20000000,16411
Dhaka,Bangladesh,21000000,306
Osaka,Japan,19200000,225
New York,USA,18800000,783
Lagos,Nigeria,15400000,1171
`

func newModel(st store, path string) (model, error) {
	in := textinput.New()
	in.Width = 40
	m := model{st: st, path: path, sortCol: -1, in: in}
	text := sample
	if path != "" {
		t, err := st.read(path)
		if err != nil {
			return m, err
		}
		text = t
	}
	s, err := parseCSV(text)
	if err != nil {
		return m, err
	}
	m.cur, m.saved = s, s.encode()
	if path != "" {
		m.msg = "Opened " + path
	}
	return m, nil
}

func (m model) Init() tui.Cmd { return nil }

func (m model) dirty() bool { return m.cur.encode() != m.saved }

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) visRows() int { return max(1, m.size().H-chrome) }

func (m model) gutter() int { return len(strconv.Itoa(max(1, len(m.cur.rows)))) + 1 }

func (m model) colW(c int) int {
	w := ansi.Width(m.cur.headers[c]) + 2
	for _, r := range m.cur.rows {
		w = max(w, ansi.Width(r[c])+1)
	}
	return max(minColW, min(maxColW, w))
}

// span is a column and where it is drawn.
type span struct{ col, x, w int }

// visibleCols are the columns from m.left that fit, the last one cut short if
// it does not.
func (m model) visibleCols() []span {
	var out []span
	x, limit := m.gutter(), m.size().W
	for c := m.left; c < m.cur.cols() && x < limit; c++ {
		w := min(m.colW(c), limit-x)
		out = append(out, span{c, x, w})
		x += m.colW(c)
	}
	return out
}

// keep scrolls so the cursor is on screen.
func (m model) keep() model {
	m.row = max(-1, min(m.row, len(m.cur.rows)-1))
	m.col = max(0, min(m.col, m.cur.cols()-1))
	if m.row >= 0 {
		m.top = max(0, min(m.top, m.row))
		if m.row >= m.top+m.visRows() {
			m.top = m.row - m.visRows() + 1
		}
	}
	m.top = max(0, min(m.top, max(0, len(m.cur.rows)-m.visRows())))
	m.left = min(m.left, m.col)
	for m.left < m.col && !fits(m.visibleCols(), m.col, m.colW(m.col), m.size().W) {
		m.left++
	}
	return m
}

// fits reports whether column c is drawn whole.
func fits(vc []span, c, w, limit int) bool {
	for _, s := range vc {
		if s.col == c {
			return s.x+w <= limit
		}
	}
	return false
}

// ---- editing ---------------------------------------------------------------

// change replaces the sheet, keeping the old one for undo.
func (m model) change(s sheet) model {
	m.undo = append(m.undo, m.cur)
	if len(m.undo) > undoMax {
		m.undo = m.undo[1:]
	}
	m.redo, m.cur, m.sortCol = nil, s, -1
	return m.keep()
}

func (m model) undoStep() model {
	if len(m.undo) == 0 {
		m.msg = "Nothing to undo"
		return m
	}
	m.redo = append(m.redo, m.cur)
	m.cur, m.undo = m.undo[len(m.undo)-1], m.undo[:len(m.undo)-1]
	m.sortCol, m.msg = -1, "Undone"
	return m.keep()
}

func (m model) redoStep() model {
	if len(m.redo) == 0 {
		m.msg = "Nothing to redo"
		return m
	}
	m.undo = append(m.undo, m.cur)
	m.cur, m.redo = m.redo[len(m.redo)-1], m.redo[:len(m.redo)-1]
	m.sortCol, m.msg = -1, "Redone"
	return m.keep()
}

func (m model) ref() string {
	if m.row < 0 {
		return colName(m.col) + "0"
	}
	return colName(m.col) + strconv.Itoa(m.row+1)
}

func (m model) startEdit() model {
	m.mode = modeEdit
	m.in.Prompt = m.ref() + " ▸ "
	m.in.SetValue(m.cur.cell(m.row, m.col))
	m.in.CursorEnd()
	m.in.Focus()
	return m
}

func (m model) prompt(md mode, label, initial string) model {
	m.mode = md
	m.in.Prompt = label
	m.in.SetValue(initial)
	m.in.CursorEnd()
	m.in.Focus()
	return m
}

func (m model) closeInput() model {
	m.mode = modeGrid
	m.in.Blur()
	return m
}

func (m model) sortBy(c int) model {
	desc := m.sortCol == c && !m.sortDesc
	m = m.change(m.cur.sorted(c, desc))
	m.sortCol, m.sortDesc = c, desc
	dir := "ascending"
	if desc {
		dir = "descending"
	}
	m.msg = "Sorted by " + m.cur.headers[c] + ", " + dir
	m.row = max(m.row, 0)
	return m.keep()
}

func (m model) insertRow(below bool) model {
	i := m.row
	if m.row < 0 {
		i = 0
	} else if below {
		i = m.row + 1
	}
	m = m.change(m.cur.insertRow(i))
	m.row = i
	m.msg = "Row added"
	return m.keep()
}

func (m model) deleteRow() model {
	if m.row < 0 || len(m.cur.rows) == 0 {
		m.msg = "Select a row to delete"
		return m
	}
	m = m.change(m.cur.deleteRow(m.row))
	m.msg = "Row deleted"
	return m.keep()
}

func (m model) deleteCol() model {
	s, ok := m.cur.deleteCol(m.col)
	if !ok {
		m.msg = "The last column cannot be deleted"
		return m
	}
	name := m.cur.headers[m.col]
	m = m.change(s)
	m.msg = "Column " + name + " deleted"
	return m.keep()
}

func (m model) save(path string) model {
	if err := m.st.write(path, m.cur.encode()); err != nil {
		m.msg = "Cannot save: " + err.Error()
		return m
	}
	m.path, m.saved, m.msg = path, m.cur.encode(), "Saved "+path
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.keep(), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	switch m.mode {
	case modeDiscard:
		return m.discardKey(k)
	case modeEdit, modeNewCol, modeSaveAs:
		return m.inputKey(k)
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	m.msg = ""
	if k.Type == tui.KeyCtrl {
		return m.ctrlKey(k.Code)
	}
	if next, ok := m.moveKey(k); ok {
		return next.keep(), nil
	}
	return m.actionKey(k)
}

func (m model) ctrlKey(code rune) (tui.Model, tui.Cmd) {
	switch code {
	case 's':
		if m.path == "" {
			return m.prompt(modeSaveAs, "Save as ▸ ", ""), nil
		}
		return m.save(m.path), nil
	case 'z':
		return m.undoStep(), nil
	case 'y':
		return m.redoStep(), nil
	case 'q':
		return m.quit()
	}
	return m, nil
}

func (m model) quit() (tui.Model, tui.Cmd) {
	if m.dirty() {
		m.mode = modeDiscard
		return m, nil
	}
	return m, tui.Quit()
}

func (m model) moveKey(k tui.Key) (model, bool) {
	page := max(1, m.visRows()-1)
	switch k.Type {
	case tui.KeyUp:
		m.row--
	case tui.KeyDown:
		m.row++
	case tui.KeyLeft:
		m.col--
	case tui.KeyRight, tui.KeyTab:
		m.col++
	case tui.KeyPgUp:
		m.row -= page
	case tui.KeyPgDown:
		m.row += page
	case tui.KeyHome:
		m.col = 0
	case tui.KeyEnd:
		m.col = m.cur.cols() - 1
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.row--
		case "j":
			m.row++
		case "h":
			m.col--
		case "l":
			m.col++
		case "g":
			m.row = -1
		case "G":
			m.row = len(m.cur.rows) - 1
		default:
			return m, false
		}
	default:
		return m, false
	}
	return m, true
}

func (m model) actionKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEnter, tui.KeyF2:
		return m.startEdit(), nil
	case tui.KeyDelete:
		return m.deleteRow(), nil
	case tui.KeyEsc:
		return m, nil
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "e":
		return m.startEdit(), nil
	case "o":
		return m.insertRow(true), nil
	case "O":
		return m.insertRow(false), nil
	case "d":
		return m.deleteRow(), nil
	case "+", "=":
		return m.prompt(modeNewCol, "New column ▸ ", ""), nil
	case "-":
		return m.deleteCol(), nil
	case "s":
		return m.sortBy(m.col), nil
	case "c":
		m.msg = "Copied " + m.ref() + " (if your terminal allows it)"
		return m, tui.WriteClipboard(m.cur.cell(m.row, m.col))
	case "u":
		return m.undoStep(), nil
	case "q":
		return m.quit()
	}
	return m, nil
}

func (m model) discardKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && (k.Text == "y" || k.Text == "Y"):
		return m, tui.Quit()
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && (k.Text == "n" || k.Text == "N"):
		m.mode = modeGrid
	}
	return m, nil
}

func (m model) inputKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.msg = "Cancelled"
		return m.closeInput(), nil
	case tui.KeyEnter:
		return m.commit(), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

func (m model) commit() model {
	v := m.in.Value()
	md := m.mode
	m = m.closeInput()
	switch md {
	case modeEdit:
		if v == m.cur.cell(m.row, m.col) {
			return m
		}
		m = m.change(m.cur.set(m.row, m.col, v))
		m.msg = "Set " + m.ref()
	case modeNewCol:
		name := strings.TrimSpace(v)
		if name == "" {
			m.msg = "A column needs a name"
			return m
		}
		m = m.change(m.cur.insertCol(m.col+1, name))
		m.col++
		m.msg = "Column " + name + " added"
	case modeSaveAs:
		if p := strings.TrimSpace(v); p != "" {
			m = m.save(p)
		}
	}
	return m.keep()
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

var toolbar = []struct{ id, label string }{
	{"tb-row", "+Row"}, {"tb-col", "+Col"}, {"tb-delrow", "−Row"}, {"tb-delcol", "−Col"},
	{"tb-sort", "Sort"}, {"tb-undo", "Undo"}, {"tb-redo", "Redo"}, {"tb-save", "Save"},
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.mode != modeGrid || m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = m.toolbarHit(ev.X, ev.Y)
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.top = max(0, m.top-3)
		m.row = max(-1, min(m.row, m.top+m.visRows()-1))
		return m.keepAfterWheel(), nil
	case tui.MouseButtonWheelDown:
		m.top = min(max(0, len(m.cur.rows)-m.visRows()), m.top+3)
		m.row = max(m.row, m.top)
		return m.keepAfterWheel(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	if id := m.toolbarHit(ev.X, ev.Y); id != "" {
		return m.tool(id)
	}
	return m.clickCell(ev)
}

// keepAfterWheel clamps without moving the view back to the cursor.
func (m model) keepAfterWheel() model {
	top := m.top
	m = m.keep()
	m.top = top
	return m
}

func (m model) toolbarHit(x, y int) string {
	for _, t := range toolbar {
		if r, ok := layout.RectOf(m.screen(), m.size(), t.id); ok && inside(r, x, y) {
			return t.id
		}
	}
	return ""
}

func (m model) tool(id string) (tui.Model, tui.Cmd) {
	switch id {
	case "tb-row":
		return m.insertRow(true), nil
	case "tb-col":
		return m.prompt(modeNewCol, "New column ▸ ", ""), nil
	case "tb-delrow":
		return m.deleteRow(), nil
	case "tb-delcol":
		return m.deleteCol(), nil
	case "tb-sort":
		return m.sortBy(m.col), nil
	case "tb-undo":
		return m.undoStep(), nil
	case "tb-redo":
		return m.redoStep(), nil
	}
	return m.ctrlKey('s')
}

// cellAt is the cell drawn at x, y: row -1 for the header, ok false elsewhere.
func (m model) cellAt(x, y int) (row, col int, ok bool) {
	g, found := layout.RectOf(m.screen(), m.size(), "grid")
	if !found || !inside(g, x, y) {
		return 0, 0, false
	}
	line := y - g.Y // 0 is the header
	row = m.top + line - 1
	if line == 0 {
		row = -1
	}
	if row >= len(m.cur.rows) {
		return 0, 0, false
	}
	for _, s := range m.visibleCols() {
		if x >= s.x && x < s.x+s.w {
			return row, s.col, true
		}
	}
	return 0, 0, false
}

func (m model) clickCell(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	row, col, ok := m.cellAt(ev.X, ev.Y)
	if !ok {
		return m, nil
	}
	same := row == m.row && col == m.col
	m.row, m.col = row, col
	switch {
	case row == -1 && same:
		return m.sortBy(col), nil
	case ev.Clicks >= 2 || (same && row >= 0 && ev.Clicks == 0):
		return m.startEdit(), nil
	}
	top := m.top
	m = m.keep()
	m.top = top
	return m, nil
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) titleLine(th theme.Theme) string {
	name := m.path
	if name == "" {
		name = "sample"
	}
	if m.dirty() {
		name += " ●"
	}
	info := fmt.Sprintf("%d rows × %d cols", len(m.cur.rows), m.cur.cols())
	w := m.size().W
	left := ansi.NewStyle().Bold().Foreground(th.Primary).Render("csvedit") + "  " + ansi.NewStyle().Bold().Render(name)
	gap := max(1, w-ansi.Width(left)-ansi.Width(info)-1)
	return ansi.Truncate(left+strings.Repeat(" ", gap)+ansi.NewStyle().Faint().Render(info), w)
}

func (m model) toolbarNode(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	for _, t := range toolbar {
		st := ansi.NewStyle().Foreground(th.Muted)
		off := (t.id == "tb-undo" && len(m.undo) == 0) || (t.id == "tb-redo" && len(m.redo) == 0)
		switch {
		case off:
			st = ansi.NewStyle().Faint()
		case m.hover == t.id:
			st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(t.id, layout.Block(st.Render("["+t.label+"]")))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) cellText(r, c, w int, th theme.Theme, right bool) string {
	text := ansi.Truncate(m.cur.cell(r, c), w-1)
	if r < 0 {
		if m.sortCol == c {
			arrow := "▲"
			if m.sortDesc {
				arrow = "▼"
			}
			text = ansi.Truncate(m.cur.cell(r, c), w-3) + " " + arrow
		}
		text = ansi.NewStyle().Bold().Render(text)
	}
	pw := w
	if right && r >= 0 {
		return strings.Repeat(" ", max(0, pw-1-ansi.Width(text))) + text + " "
	}
	return pad(text, pw)
}

func (m model) gridNode(th theme.Theme) layout.Node {
	size := m.size()
	vc := m.visibleCols()
	var lines []layout.FlexChild
	build := func(r int) string {
		var b strings.Builder
		gut := ""
		if r >= 0 {
			gut = strconv.Itoa(r + 1)
		}
		b.WriteString(ansi.NewStyle().Faint().Render(pad(strings.Repeat(" ", m.gutter()-1-len(gut))+gut, m.gutter())))
		for _, s := range vc {
			text := m.cellText(r, s.col, s.w, th, m.cur.numeric(s.col))
			text = ansi.Truncate(text, s.w)
			switch {
			case r == m.row && s.col == m.col:
				text = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary).Render(ansi.StripANSI(text))
			case r == m.row:
				text = ansi.NewStyle().Background(th.Selection).Render(ansi.StripANSI(text))
			case r < 0:
				text = ansi.NewStyle().Underline().Render(ansi.StripANSI(text))
			}
			b.WriteString(pad(text, s.w))
		}
		return ansi.Truncate(b.String(), size.W)
	}
	lines = append(lines, layout.FlexChild{Node: layout.Block(build(-1))})
	for i := 0; i < m.visRows(); i++ {
		r := m.top + i
		switch {
		case r < len(m.cur.rows):
			lines = append(lines, layout.FlexChild{Node: layout.Block(build(r))})
		case i == 0 && len(m.cur.rows) == 0:
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(" No rows. Press o to add one."))})
		default:
			lines = append(lines, layout.FlexChild{Node: layout.Block("")})
		}
	}
	return layout.Named("grid", layout.Fixed(layout.Column(0, lines...), layout.Size{W: size.W, H: m.visRows() + 1}))
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	switch m.mode {
	case modeDiscard:
		return ansi.NewStyle().Bold().Foreground(th.Warning).Render("Discard unsaved changes? ") + "[y] yes  [n] no"
	case modeEdit, modeNewCol, modeSaveAs:
		return ansi.Truncate(m.in.View(), w)
	}
	cell := m.cur.cell(m.row, m.col)
	left := ansi.NewStyle().Bold().Render(m.ref()) + " " + ansi.NewStyle().Faint().Render(m.cur.headers[m.col])
	if m.msg != "" {
		left += "  " + m.msg
	} else if cell != "" {
		left += "  " + ansi.NewStyle().Faint().Render(strconv.Quote(cell))
	}
	return ansi.Truncate(left, w)
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate("  "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16))+a, m.size().W))}
	}
	lines := []layout.FlexChild{
		key("arrows hjkl", "move"), key("Enter F2 e", "edit the cell (Enter saves)"), key("o O", "add a row below or above"),
		key("d Delete", "delete the row"), key("+ -", "add or delete a column"), key("s", "sort by the column"),
		key("Ctrl+Z Ctrl+Y", "undo and redo"), key("c", "copy the cell"), key("Ctrl+S", "save"),
		key("g G", "first row, last row"), key("q Ctrl+Q", "quit, asking first"), key("F1", "close help"),
	}
	return layout.Fixed(layout.Column(0, lines...), layout.Size{W: m.size().W, H: m.visRows() + 1})
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	body := m.gridNode(th)
	if m.help {
		body = m.helpNode(th)
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "enter", Action: "edit"}, widgets.Hint{Key: "o", Action: "row"},
		widgets.Hint{Key: "s", Action: "sort"}, widgets.Hint{Key: "ctrl+z", Action: "undo"},
		widgets.Hint{Key: "ctrl+s", Action: "save"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(m.titleLine(th))},
		layout.FlexChild{Node: m.toolbarNode(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	m, err := newModel(diskStore{}, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "csvedit:", err)
		os.Exit(1)
	}
	if _, err := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
