// Command sqlbrowse is a read-only SQL browser over a small made-up shop
// database held in memory: pick a table, write a SELECT, read the result in a
// grid, export it as CSV, and recall earlier queries. There is no database
// driver and nothing is read from or written to disk (unless you give -out).
// The SQL is a small subset: SELECT with WHERE, JOIN (one), GROUP BY, HAVING,
// ORDER BY, LIMIT/OFFSET, DISTINCT, COUNT, SUM, AVG, MIN, MAX and a few functions.
//
// Mouse: click a table to look inside it, click the query to type, click a
// result cell, scroll the grid, click Run, History, Tables or Export. Keyboard:
// type a query and press Enter (or F5) to run it; Up and Down in the query
// recall earlier ones; Tab moves between the query, the result and the table
// list; Ctrl+R opens the history, Ctrl+T the tables, Ctrl+E copies the result
// as CSV, Ctrl+L clears the query; in the result the arrows move, y copies the
// cell, e exports; F1 shows help; Ctrl+C quits (q too outside the query).
//
//	go run ./examples/sqlbrowse
//	go run ./examples/sqlbrowse -out result.csv    # Export also writes this file
package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	sideW      = 26
	wideAt     = 72
	chrome     = 3 // title, status, hints
	maxShown   = 10000
	maxHistory = 100
	maxColW    = 28
)

type focus int

const (
	onQuery focus = iota
	onGrid
	onTables
)

type pick int

const (
	noPick pick = iota
	pickTables
	pickHistory
)

type model struct {
	db  *database
	now func() time.Time

	in      textinput.Model
	focus   focus
	history []string // oldest first
	pick    pick
	pickCur int

	res       *result
	truncated bool
	err       *sqlError
	ran       string // the query that gave res or err
	took      time.Duration

	gr, gc, gtop, gleft int
	tCur                int

	out   string // file Export also writes
	msg   string
	help  bool
	hover string

	width, height int
}

// newModel opens db with a first query already run. now is the clock queries
// are timed with; nil means the real one.
func newModel(db *database, out string, now func() time.Time) model {
	in := textinput.New()
	in.Prompt = "SQL ▸ "
	in.Placeholder = "SELECT … FROM …"
	in.Focus()
	if now == nil {
		now = time.Now
	}
	const first = "SELECT * FROM customers LIMIT 20"
	in.SetValue(first)
	return model{db: db, now: now, in: in, out: out}.run(first)
}

func (m model) Init() tui.Cmd { return nil }

// ---- running a query -------------------------------------------------------

func (m model) run(q string) model {
	q = strings.TrimSpace(q)
	if q == "" {
		m.msg = "Type a query first"
		return m
	}
	if n := len(m.history); n == 0 || m.history[n-1] != q {
		m.history = append(m.history, q)
		if len(m.history) > maxHistory {
			m.history = m.history[1:]
		}
	}
	m.in.History = m.history
	began := m.now()
	res, err := m.db.query(q)
	m.took = m.now().Sub(began)
	m.ran, m.err, m.res, m.truncated = q, err, res, false
	m.gr, m.gc, m.gtop, m.gleft, m.msg = 0, 0, 0, 0, ""
	if res != nil && len(res.rows) > maxShown {
		res.rows, m.truncated = res.rows[:maxShown], true
	}
	return m.keepGrid()
}

// csvOf is the result as CSV, NULL as an empty field.
func (r *result) csvOf() string {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write(r.cols)
	for _, row := range r.rows {
		rec := make([]string, len(row))
		for i, v := range row {
			if v != nil {
				rec[i] = format(v)
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	return b.String()
}

func writeFile(path, text string) tui.Cmd {
	return func() tui.Msg {
		_ = os.WriteFile(path, []byte(text), 0o600)
		return nil
	}
}

func (m model) export() (tui.Model, tui.Cmd) {
	if m.res == nil {
		m.msg = "Run a query that works first"
		return m, nil
	}
	text := m.res.csvOf()
	m.msg = fmt.Sprintf("Copied %d rows as CSV (if your terminal allows it)", len(m.res.rows))
	cmd := tui.WriteClipboard(text)
	if m.out != "" {
		m.msg = fmt.Sprintf("Copied %d rows as CSV and wrote %s", len(m.res.rows), m.out)
		cmd = tui.Batch(cmd, writeFile(m.out, text))
	}
	return m, cmd
}

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 28}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool { return m.size().W >= wideAt }

func (m model) rightW() int {
	if m.wide() {
		return m.size().W - sideW - 1
	}
	return m.size().W
}

// gridRows is how many result rows fit under the heading.
func (m model) gridRows() int { return max(1, m.size().H-chrome-2-1) }

func (m model) gutter() int {
	if m.res == nil {
		return 0
	}
	return len(strconv.Itoa(max(1, len(m.res.rows)))) + 1
}

func (m model) colWidths() []int {
	if m.res == nil {
		return nil
	}
	ws := make([]int, len(m.res.cols))
	for i, c := range m.res.cols {
		ws[i] = utf8.RuneCountInString(c)
	}
	for _, r := range m.res.rows[:min(len(m.res.rows), 200)] {
		for i, v := range r {
			ws[i] = max(ws[i], utf8.RuneCountInString(format(v)))
		}
	}
	for i := range ws {
		ws[i] = max(3, min(ws[i], maxColW)) + 2
	}
	return ws
}

type span struct{ col, x, w int }

// spans are the columns from gleft that fit, with where each is drawn (x counts
// from the left edge of the result area).
func (m model) spans() []span {
	ws := m.colWidths()
	var out []span
	x := m.gutter()
	for c := m.gleft; c < len(ws) && x < m.rightW(); c++ {
		out = append(out, span{c, x, min(ws[c], m.rightW()-x)})
		x += ws[c]
	}
	return out
}

func (m model) keepGrid() model {
	if m.res == nil {
		return m
	}
	m.gr = max(0, min(m.gr, len(m.res.rows)-1))
	m.gc = max(0, min(m.gc, len(m.res.cols)-1))
	if m.gr < m.gtop {
		m.gtop = m.gr
	}
	if m.gr >= m.gtop+m.gridRows() {
		m.gtop = m.gr - m.gridRows() + 1
	}
	m.gtop = max(0, min(m.gtop, max(0, len(m.res.rows)-m.gridRows())))
	m.gleft = min(m.gleft, m.gc)
	ws := m.colWidths()
	for m.gleft < m.gc {
		used := m.gutter()
		for c := m.gleft; c <= m.gc; c++ {
			used += ws[c]
		}
		if used <= m.rightW() {
			break
		}
		m.gleft++
	}
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.keepGrid(), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(msg) // the cursor blink
	return m, cmd
}

func (m model) setFocus(f focus) (model, tui.Cmd) {
	if f == onTables && !m.wide() {
		f = onQuery
	}
	m.focus = f
	if f == onQuery {
		return m, m.in.Focus()
	}
	m.in.Blur()
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	if m.pick != noPick {
		return m.pickKey(k)
	}
	m.msg = ""
	if k.Type == tui.KeyCtrl {
		if next, cmd, ok := m.ctrlKey(k.Code); ok {
			return next, cmd
		}
	}
	if k.Type == tui.KeyF5 {
		return m.run(m.in.Value()), nil
	}
	if k.Type == tui.KeyTab {
		return m.cycle(k.Mod.Shift())
	}
	switch m.focus {
	case onGrid:
		return m.gridKey(k)
	case onTables:
		return m.tablesKey(k)
	}
	return m.queryKey(k)
}

func (m model) ctrlKey(code rune) (model, tui.Cmd, bool) {
	switch code {
	case 'r':
		m.pick, m.pickCur = pickHistory, 0
		return m, nil, true
	case 't':
		m.pick, m.pickCur = pickTables, m.tCur
		return m, nil, true
	case 'e':
		next, cmd := m.export()
		return next.(model), cmd, true
	case 'l':
		m.in.Reset()
		return m, nil, true
	}
	return m, nil, false
}

func (m model) cycle(back bool) (tui.Model, tui.Cmd) {
	order := []focus{onQuery, onGrid}
	if m.wide() {
		order = append(order, onTables)
	}
	at := 0
	for i, f := range order {
		if f == m.focus {
			at = i
		}
	}
	d := 1
	if back {
		d = len(order) - 1
	}
	return m.setFocus(order[(at+d)%len(order)])
}

func (m model) queryKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEnter:
		return m.run(m.in.Value()), nil
	case tui.KeyEsc:
		return m.setFocus(onGrid)
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

func (m model) gridKey(k tui.Key) (tui.Model, tui.Cmd) {
	page := max(1, m.gridRows()-1)
	switch k.Type {
	case tui.KeyUp:
		m.gr--
	case tui.KeyDown:
		m.gr++
	case tui.KeyLeft:
		m.gc--
	case tui.KeyRight:
		m.gc++
	case tui.KeyPgUp:
		m.gr -= page
	case tui.KeyPgDown:
		m.gr += page
	case tui.KeyHome:
		m.gc = 0
	case tui.KeyEnd:
		if m.res != nil {
			m.gc = len(m.res.cols) - 1
		}
	case tui.KeyEnter, tui.KeyEsc:
		return m.setFocus(onQuery)
	case tui.KeyRunes:
		return m.gridRune(k.Text)
	}
	return m.keepGrid(), nil
}

func (m model) gridRune(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "k":
		m.gr--
	case "j":
		m.gr++
	case "h":
		m.gc--
	case "l":
		m.gc++
	case "g":
		m.gr = 0
	case "G":
		if m.res != nil {
			m.gr = len(m.res.rows) - 1
		}
	case "i", "/":
		return m.setFocus(onQuery)
	case "q":
		return m, tui.Quit()
	case "?":
		m.help = true
	case "e":
		return m.export()
	case "y":
		return m.copyCell()
	}
	return m.keepGrid(), nil
}

func (m model) copyCell() (tui.Model, tui.Cmd) {
	if m.res == nil || m.gr >= len(m.res.rows) {
		m.msg = "No cell to copy"
		return m, nil
	}
	v := format(m.res.rows[m.gr][m.gc])
	m.msg = "Copied " + strconv.Quote(ansi.Truncate(v, 40)) + " (if your terminal allows it)"
	return m, tui.WriteClipboard(v)
}

func (m model) tablesKey(k tui.Key) (tui.Model, tui.Cmd) {
	n := len(m.db.tables)
	switch k.Type {
	case tui.KeyUp:
		m.tCur = max(0, m.tCur-1)
	case tui.KeyDown:
		m.tCur = min(n-1, m.tCur+1)
	case tui.KeyEnter:
		return m.openTable(m.tCur), nil
	case tui.KeyEsc:
		return m.setFocus(onQuery)
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.tCur = max(0, m.tCur-1)
		case "j":
			m.tCur = min(n-1, m.tCur+1)
		case "q":
			return m, tui.Quit()
		}
	}
	return m, nil
}

// openTable writes a query for table i into the box and runs it.
func (m model) openTable(i int) model {
	q := "SELECT * FROM " + m.db.tables[i].name + " LIMIT 100"
	m.tCur = i
	m.in.SetValue(q)
	m.in.CursorEnd()
	return m.run(q)
}

func (m model) items() []string {
	if m.pick == pickTables {
		var out []string
		for _, t := range m.db.tables {
			out = append(out, fmt.Sprintf("%s  (%d rows)", t.name, len(t.rows)))
		}
		return out
	}
	out := make([]string, len(m.history))
	for i, q := range m.history {
		out[len(m.history)-1-i] = strings.Join(strings.Fields(q), " ") // newest first
	}
	return out
}

func (m model) pickKey(k tui.Key) (tui.Model, tui.Cmd) {
	n := len(m.items())
	switch k.Type {
	case tui.KeyEsc:
		m.pick = noPick
	case tui.KeyUp:
		m.pickCur = max(0, m.pickCur-1)
	case tui.KeyDown:
		m.pickCur = min(n-1, m.pickCur+1)
	case tui.KeyEnter:
		return m.choose(m.pickCur)
	}
	return m, nil
}

func (m model) choose(i int) (tui.Model, tui.Cmd) {
	kind := m.pick
	m.pick = noPick
	switch {
	case kind == pickTables && i >= 0 && i < len(m.db.tables):
		return m.openTable(i), nil
	case kind == pickHistory && i >= 0 && i < len(m.history):
		m.in.SetValue(m.history[len(m.history)-1-i])
		m.in.CursorEnd()
		return m.setFocus(onQuery)
	}
	return m, nil
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"query", "run", "hist", "tables", "export", "grid"}
	for i := range m.db.tables {
		ids = append(ids, "tbl-"+strconv.Itoa(i))
	}
	for i := range m.items() {
		ids = append(ids, "ovl-"+strconv.Itoa(i))
	}
	return ids
}

func (m model) hit(x, y int) string {
	for _, id := range m.ids() {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		d := 3
		if ev.Button == tui.MouseButtonWheelUp {
			d = -3
		}
		if m.pick != noPick {
			m.pickCur = max(0, min(len(m.items())-1, m.pickCur+d/3))
			return m, nil
		}
		m.gr += d
		return m.keepGrid(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	return m.click(id, ev)
}

func (m model) click(id string, ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	n := 0
	if i := strings.LastIndexByte(id, '-'); i >= 0 {
		n, _ = strconv.Atoi(id[i+1:])
	}
	switch {
	case strings.HasPrefix(id, "ovl-"):
		return m.choose(n)
	case m.pick != noPick:
		m.pick = noPick // a click outside the list closes it
		return m, nil
	case id == "query":
		return m.setFocus(onQuery)
	case id == "run":
		return m.run(m.in.Value()), nil
	case id == "hist":
		m.pick, m.pickCur = pickHistory, 0
	case id == "tables":
		m.pick, m.pickCur = pickTables, m.tCur
	case id == "export":
		return m.export()
	case strings.HasPrefix(id, "tbl-"):
		m.focus = onTables
		m.in.Blur()
		return m.openTable(n), nil
	case id == "grid":
		return m.clickCell(ev)
	}
	return m, nil
}

func (m model) clickCell(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	r, ok := layout.RectOf(m.screen(), m.size(), "grid")
	if !ok || m.res == nil {
		return m, nil
	}
	line := ev.Y - r.Y - 1 // the first line is the heading
	if line < 0 || m.gtop+line >= len(m.res.rows) {
		return m, nil
	}
	for _, s := range m.spans() {
		if ev.X-r.X >= s.x && ev.X-r.X < s.x+s.w {
			m.gr, m.gc = m.gtop+line, s.col
			next, cmd := m.setFocus(onGrid)
			top := next.gtop
			next = next.keepGrid()
			next.gtop = top
			return next, cmd
		}
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) button(id, label string, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	if id == "run" {
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	if m.hover == id {
		st = st.Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+label+" ")))}
}

func (m model) titleNode(th theme.Theme) layout.Node {
	btns := []layout.FlexChild{m.button("run", "Run", th), m.button("hist", "History", th), m.button("export", "Export", th)}
	if !m.wide() {
		btns = append([]layout.FlexChild{m.button("tables", "Tables", th)}, btns...)
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("sqlbrowse") + ansi.NewStyle().Faint().Render("  read-only")
	return layout.Fixed(layout.Row(1, append([]layout.FlexChild{{Node: layout.Block(title), Grow: 1}}, btns...)...), layout.Size{W: m.size().W, H: 1})
}

func (m model) queryNode(th theme.Theme) layout.Node {
	in := m.in
	in.Width = max(8, m.rightW()-ansi.Width(in.Prompt)-1)
	line := ansi.Truncate(in.View(), m.rightW())
	if m.focus != onQuery {
		line = ansi.NewStyle().Faint().Render(ansi.Truncate(ansi.StripANSI(line), m.rightW()))
	}
	return layout.Named("query", layout.Fixed(layout.Block(line), layout.Size{W: m.rightW(), H: 1}))
}

func (m model) infoLine(th theme.Theme) string {
	w := m.rightW()
	switch {
	case m.err != nil:
		at := ""
		if m.err.pos >= 0 && m.err.pos <= len(m.ran) {
			at = fmt.Sprintf(" (at column %d)", utf8.RuneCountInString(m.ran[:m.err.pos])+1)
		}
		return ansi.NewStyle().Foreground(th.Error).Render(ansi.Truncate("✗ "+sentence(m.err.msg)+at, w))
	case m.res != nil:
		s := fmt.Sprintf("✓ %d rows × %d columns in %s", len(m.res.rows), len(m.res.cols), m.took.Round(10*time.Microsecond))
		if m.truncated {
			s += fmt.Sprintf(" (first %d shown)", maxShown)
		}
		return ansi.NewStyle().Foreground(th.Success).Render(ansi.Truncate(s, w))
	}
	return ansi.NewStyle().Faint().Render("Type a query and press Enter.")
}

func sentence(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (m model) gridNode(th theme.Theme) layout.Node {
	w, h := m.rightW(), m.gridRows()+1
	var kids []layout.FlexChild
	if m.res == nil {
		for i := 0; i < h; i++ {
			kids = append(kids, block(""))
		}
		return layout.Named("grid", layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: h}))
	}
	spans := m.spans()
	g := m.gutter()
	head := strings.Repeat(" ", g)
	for _, s := range spans {
		head += pad(ansi.Truncate(" "+m.res.cols[s.col], s.w), s.w)
	}
	kids = append(kids, block(ansi.NewStyle().Bold().Underline().Render(pad(ansi.Truncate(head, w), w))))
	for i := 0; i < h-1; i++ {
		idx := m.gtop + i
		if idx >= len(m.res.rows) {
			if i == 0 && len(m.res.rows) == 0 {
				kids = append(kids, block(ansi.NewStyle().Faint().Render(" (no rows)")))
				continue
			}
			kids = append(kids, block(""))
			continue
		}
		kids = append(kids, block(pad(ansi.Truncate(m.rowText(idx, spans, th), w), w)))
	}
	return layout.Named("grid", layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: h}))
}

func (m model) rowText(idx int, spans []span, th theme.Theme) string {
	row := m.res.rows[idx]
	line := ansi.NewStyle().Faint().Render(fmt.Sprintf("%*d ", m.gutter()-1, idx+1))
	for _, s := range spans {
		v := row[s.col]
		text := format(v)
		if v == nil {
			text = "NULL"
		}
		text = ansi.Truncate(text, s.w-2)
		cell := " " + text + strings.Repeat(" ", max(0, s.w-1-ansi.Width(text)))
		if _, num := v.(float64); num {
			cell = strings.Repeat(" ", max(0, s.w-1-ansi.Width(text))) + text + " "
		}
		st := ansi.NewStyle()
		switch {
		case v == nil:
			st = ansi.NewStyle().Faint().Italic()
		case idx == m.gr && s.col == m.gc && m.focus == onGrid:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		}
		if idx == m.gr && s.col == m.gc && m.focus != onGrid {
			st = st.Underline()
		}
		if idx == m.gr && s.col != m.gc {
			st = st.Background(th.Selection)
		}
		line += st.Render(cell)
	}
	return line
}

func (m model) sideNode(th theme.Theme) layout.Node {
	h := m.size().H - chrome
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	var kids []layout.FlexChild
	row := func(s string) { kids = append(kids, block(pad(ansi.Truncate(s, sideW-1), sideW-1)+rule)) }
	row(ansi.NewStyle().Bold().Render(" Tables"))
	for i, t := range m.db.tables {
		id := "tbl-" + strconv.Itoa(i)
		label := fmt.Sprintf(" %s %d", t.name, len(t.rows))
		st := ansi.NewStyle()
		switch {
		case i == m.tCur && m.focus == onTables:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case i == m.tCur:
			st = ansi.NewStyle().Bold()
		case m.hover == id:
			st = ansi.NewStyle().Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(pad(ansi.Truncate(label, sideW-1), sideW-1))+rule))})
	}
	row("")
	t := m.db.tables[m.tCur]
	row(ansi.NewStyle().Bold().Render(" " + t.name))
	for i, c := range t.cols {
		typ := ""
		if i < len(t.types) {
			typ = " " + t.types[i]
		}
		row(ansi.NewStyle().Faint().Render("  " + c + typ))
	}
	for len(kids) < h {
		row("")
	}
	return layout.Fixed(layout.Column(0, kids[:min(len(kids), h)]...), layout.Size{W: sideW, H: h})
}

func (m model) overlayNode(th theme.Theme) layout.Node {
	w, h := m.rightW(), m.gridRows()+1
	title := "Tables (Enter opens one, Esc closes)"
	if m.pick == pickHistory {
		title = "History, newest first (Enter puts it in the query, Esc closes)"
	}
	kids := []layout.FlexChild{block(ansi.NewStyle().Bold().Foreground(th.Primary).Render(ansi.Truncate(title, w)))}
	items := m.items()
	if len(items) == 0 {
		kids = append(kids, block(ansi.NewStyle().Faint().Render(" Nothing here yet.")))
	}
	top := max(0, min(m.pickCur-(h-3), max(0, len(items)-(h-1))))
	for i := top; i < len(items) && len(kids) < h; i++ {
		line := "  " + items[i]
		st := ansi.NewStyle()
		if i == m.pickCur {
			line, st = "› "+items[i], ansi.NewStyle().Bold().Background(th.Selection)
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named("ovl-"+strconv.Itoa(i), layout.Block(st.Render(pad(ansi.Truncate(line, w), w))))})
	}
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: h})
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))
	}
	return layout.Column(0,
		key("Enter F5", "run the query"), key("Up Down", "recall earlier queries (in the query)"), key("Tab", "query, result, tables"),
		key("Ctrl+R", "history list"), key("Ctrl+T", "table list"), key("Ctrl+E e", "export the result as CSV"),
		key("y", "copy the cell (in the result)"), key("Ctrl+L", "clear the query"),
		key("SQL", "SELECT … FROM t [JOIN u ON …] [WHERE …] [GROUP BY …] [HAVING …] [ORDER BY …] [LIMIT n]"),
		key("", "COUNT SUM AVG MIN MAX UPPER LOWER LENGTH ROUND ABS COALESCE; LIKE IN BETWEEN IS NULL"),
		key("F1", "close help"),
	)
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	if m.msg != "" {
		return ansi.Truncate(m.msg, w)
	}
	where := [...]string{"query", "result", "tables"}[m.focus]
	return ansi.NewStyle().Faint().Render(ansi.Truncate(fmt.Sprintf("%s · %d tables · %d queries in history", where, len(m.db.tables), len(m.history)), w))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	var right layout.Node
	switch {
	case m.help:
		right = m.helpNode(th)
	case m.pick != noPick:
		right = layout.Column(0, layout.FlexChild{Node: m.queryNode(th)}, block(""), layout.FlexChild{Node: m.overlayNode(th)})
	default:
		right = layout.Column(0, layout.FlexChild{Node: m.queryNode(th)}, block(m.infoLine(th)), layout.FlexChild{Node: m.gridNode(th)})
	}
	var body layout.Node = right
	if m.wide() {
		body = layout.Row(0, layout.FlexChild{Node: m.sideNode(th)}, layout.FlexChild{Node: right})
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "enter", Action: "run"}, widgets.Hint{Key: "tab", Action: "pane"}, widgets.Hint{Key: "ctrl+r", Action: "history"},
		widgets.Hint{Key: "ctrl+t", Action: "tables"}, widgets.Hint{Key: "ctrl+e", Action: "export"}, widgets.Hint{Key: "f1", Action: "help"})
	return layout.Column(0,
		layout.FlexChild{Node: m.titleNode(th)},
		layout.FlexChild{Node: layout.Fixed(body, layout.Size{W: w, H: m.size().H - chrome})},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	out := flag.String("out", "", "also write the exported CSV to this file")
	flag.Parse()
	if _, err := tui.NewProgram(newModel(fixture(), *out, nil), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
