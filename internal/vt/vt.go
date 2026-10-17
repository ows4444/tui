// Package vt is a terminal emulator for the output of a program run on a
// pseudo-terminal: it reads bytes and keeps the screen they draw. It models what
// full-screen programs need: cursor movement, a scrolling region, insert and
// delete of lines and characters, the alternate screen, scrollback, tab stops,
// the DEC line-drawing character set, wide and combining characters, the
// colours and attributes of SGR, and the reports a program asks for (device
// attributes, cursor position).
//
// The output is untrusted, so nothing in it reaches the host terminal: titles
// are kept as sanitised text, clipboard, hyperlink and image sequences are
// skipped whole, and the replies are a fixed few. Not safe for concurrent use.
package vt

import (
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// Attr is a set of text attributes.
type Attr uint8

// The attributes of a cell.
const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	Blink
	Reverse
	Conceal
	Strike
)

// Cell is one column of the screen. A wide cluster is a head cell (Width 2)
// followed by a continuation cell (Width 0, empty Cluster).
type Cell struct {
	Cluster string
	Width   uint8
	Attrs   Attr
	FG, BG  ansi.Color // nil is the terminal's default colour
}

func blankCell(bg ansi.Color) Cell { return Cell{Cluster: " ", Width: 1, BG: bg} }

// screen is a grid with the flag of each row that wrapped into the next.
type screen struct {
	grid [][]Cell
}

// parser states.
const (
	stGround = iota
	stEsc
	stEscInter // ESC and an intermediate byte, waiting for the final one
	stCSI
	stStr    // OSC, DCS, APC, PM or SOS: a string up to BEL or ST
	stStrEsc // ESC inside a string
)

// Terminal is the emulator.
type Terminal struct {
	cols, rows int
	main, alt  screen
	useAlt     bool

	r, c     int
	wrapNext bool
	pen      Cell // attributes for the next glyph (Cluster unused)

	saved struct {
		r, c     int
		pen      Cell
		wrapNext bool
		g        [2]byte
		shift    bool
		set      bool
	}
	top, bot int // the scrolling region, first and last row

	autowrap, insertMode, curHidden bool
	appCursor, bracketed, appKeypad bool
	mouse                           MouseMode
	sgrMouse                        bool
	g                               [2]byte // character sets for G0 and G1: 'B' ASCII, '0' line drawing
	shift                           bool    // G1 is active
	tabs                            []bool

	scrollback [][]Cell
	maxBack    int

	title string
	reply func([]byte)
	last  string // the last printed cluster, for REP

	// parser
	state  int
	params []byte
	inter  []byte
	str    []byte
	strOSC bool
	utf    []byte
}

// MouseMode is which mouse events the program asked to be sent.
type MouseMode uint8

// The mouse reporting modes of xterm (?1000, ?1002, ?1003), and whether the
// SGR encoding (?1006) was asked for.
const (
	MouseNone MouseMode = iota
	MouseClick
	MouseDrag
	MouseAny
)

// New returns a blank terminal of cols x rows with the cursor at the top left
// and 1000 lines of scrollback.
func New(cols, rows int) *Terminal {
	t := &Terminal{cols: max(cols, 1), rows: max(rows, 1), autowrap: true, maxBack: 1000}
	t.g = [2]byte{'B', 'B'}
	t.main.grid = t.blankGrid(t.rows)
	t.alt.grid = t.blankGrid(t.rows)
	t.bot = t.rows - 1
	t.resetTabs()
	return t
}

// SetReply sets where replies to the program's queries go (cursor position,
// device attributes). With none set they are dropped.
func (t *Terminal) SetReply(f func([]byte)) { t.reply = f }

// SetScrollback sets how many lines scrolled off the top are kept; 0 keeps none.
func (t *Terminal) SetScrollback(n int) {
	t.maxBack = max(n, 0)
	if len(t.scrollback) > t.maxBack {
		t.scrollback = append([][]Cell(nil), t.scrollback[len(t.scrollback)-t.maxBack:]...)
	}
}

func (t *Terminal) blankGrid(n int) [][]Cell {
	g := make([][]Cell, n)
	for i := range g {
		g[i] = t.blankRow(nil)
	}
	return g
}

func (t *Terminal) blankRow(bg ansi.Color) []Cell {
	row := make([]Cell, t.cols)
	for i := range row {
		row[i] = blankCell(bg)
	}
	return row
}

func (t *Terminal) resetTabs() {
	t.tabs = make([]bool, t.cols)
	for i := 0; i < t.cols; i += 8 {
		t.tabs[i] = true
	}
}

// cur is the screen being drawn on.
func (t *Terminal) cur() *screen {
	if t.useAlt {
		return &t.alt
	}
	return &t.main
}

// Size returns the columns and rows.
func (t *Terminal) Size() (cols, rows int) { return t.cols, t.rows }

// Cell returns the cell at column x, row y of the screen; a blank cell outside it.
func (t *Terminal) Cell(x, y int) Cell {
	if x < 0 || y < 0 || x >= t.cols || y >= t.rows {
		return blankCell(nil)
	}
	return t.cur().grid[y][x]
}

// Cursor returns the cursor position and whether the program shows it.
func (t *Terminal) Cursor() (x, y int, visible bool) { return t.c, t.r, !t.curHidden }

// Title is the window title the program set, as plain text.
func (t *Terminal) Title() string { return t.title }

// AppCursorKeys reports whether the program asked for application cursor keys
// (DECCKM), which changes the bytes the arrow keys send.
func (t *Terminal) AppCursorKeys() bool { return t.appCursor }

// BracketedPaste reports whether the program asked for pasted text to be
// wrapped in markers.
func (t *Terminal) BracketedPaste() bool { return t.bracketed }

// AppKeypad reports whether the program set the application keypad.
func (t *Terminal) AppKeypad() bool { return t.appKeypad }

// AltScreen reports whether the alternate screen is showing.
func (t *Terminal) AltScreen() bool { return t.useAlt }

// Mouse is the mouse reporting the program asked for, and whether it wants the
// SGR encoding.
func (t *Terminal) Mouse() MouseMode { return t.mouse }

// MouseSGR reports whether the program asked for mouse reports in the SGR
// encoding (?1006), which is the only one the pane sends.
func (t *Terminal) MouseSGR() bool { return t.sgrMouse }

// Scrollback is the number of lines kept above the screen.
func (t *Terminal) Scrollback() int { return len(t.scrollback) }

// Row returns row y of the view scrolled back by offset lines: offset 0 is the
// live screen, and an offset of Scrollback() is the oldest line at the top. It
// returns the live screen's row on the alternate screen, which has no history.
// The slice is a copy.
func (t *Terminal) Row(offset, y int) []Cell {
	if y < 0 || y >= t.rows {
		return t.blankRow(nil)
	}
	if t.useAlt || offset <= 0 {
		return append([]Cell(nil), t.cur().grid[y]...)
	}
	offset = min(offset, len(t.scrollback))
	i := len(t.scrollback) - offset + y // an index into scrollback followed by the screen
	var src []Cell
	if i < len(t.scrollback) {
		src = t.scrollback[i]
	} else {
		src = t.main.grid[i-len(t.scrollback)]
	}
	row := t.blankRow(nil)
	copy(row, src)
	return row
}

// Resize changes the size. Rows that no longer fit above the cursor go to the
// scrollback; columns are cropped or extended with blanks, with no reflow.
func (t *Terminal) Resize(cols, rows int) {
	cols, rows = max(cols, 1), max(rows, 1)
	if cols == t.cols && rows == t.rows {
		return
	}
	if !t.useAlt && rows < t.rows && t.r > rows-1 {
		t.scrollOffTop(t.r - (rows - 1))
		t.r = rows - 1
	}
	for _, s := range []*screen{&t.main, &t.alt} {
		s.grid = resizeGrid(s.grid, cols, rows)
	}
	for i, row := range t.scrollback {
		t.scrollback[i] = resizeRow(row, cols)
	}
	t.cols, t.rows = cols, rows
	t.top, t.bot = 0, rows-1
	t.r, t.c = min(t.r, rows-1), min(t.c, cols-1)
	t.wrapNext = false
	t.resetTabs()
}

func resizeRow(row []Cell, cols int) []Cell {
	if len(row) == cols {
		return row
	}
	out := make([]Cell, cols)
	n := copy(out, row)
	for i := n; i < cols; i++ {
		out[i] = blankCell(nil)
	}
	if n > 0 && n == cols && out[n-1].Width == 2 { // a wide head cut off from its tail
		out[n-1] = blankCell(nil)
	}
	return out
}

func resizeGrid(g [][]Cell, cols, rows int) [][]Cell {
	out := make([][]Cell, rows)
	for i := range out {
		if i < len(g) {
			out[i] = resizeRow(g[i], cols)
			continue
		}
		out[i] = resizeRow(nil, cols)
	}
	return out
}

// scrollOffTop moves the top n rows of the main screen into the scrollback.
func (t *Terminal) scrollOffTop(n int) {
	n = min(n, len(t.main.grid))
	for _, row := range t.main.grid[:n] {
		t.pushBack(row)
	}
	t.main.grid = append(t.main.grid[n:], t.blankGrid(n)...)
}

func (t *Terminal) pushBack(row []Cell) {
	if t.maxBack == 0 {
		return
	}
	t.scrollback = append(t.scrollback, append([]Cell(nil), row...))
	if len(t.scrollback) > t.maxBack+t.maxBack/4+1 { // trim in batches, not on every line
		t.scrollback = append([][]Cell(nil), t.scrollback[len(t.scrollback)-t.maxBack:]...)
	}
}

// Write feeds output of the program to the terminal.
func (t *Terminal) Write(p []byte) {
	for _, b := range p {
		t.feed(b)
	}
}

func (t *Terminal) feed(b byte) {
	switch t.state {
	case stStr:
		t.feedString(b)
		return
	case stStrEsc:
		if b == '\\' {
			t.endString()
			return
		}
		t.str = nil // an ESC that did not end the string starts a sequence
		t.state = stEsc
		t.feed(b)
		return
	}
	// C0 controls act wherever they arrive, except that ESC, CAN and SUB end a
	// sequence in progress.
	switch {
	case b == 0x7f: // DEL is ignored
		return
	case b == 0x18 || b == 0x1a:
		t.state = stGround
		t.utf = nil
		return
	case b == 0x1b:
		t.state = stEsc
		t.utf = nil
		t.params, t.inter = t.params[:0], t.inter[:0]
		return
	case b < 0x20:
		t.control(b)
		return
	}
	switch t.state {
	case stGround:
		t.ground(b)
	case stEsc:
		t.escape(b)
	case stEscInter:
		t.escInter(b)
	case stCSI:
		t.csiByte(b)
	}
}

// control runs a C0 control character.
func (t *Terminal) control(b byte) {
	switch b {
	case 0x08: // BS
		t.c = max(t.c-1, 0)
		t.wrapNext = false
	case 0x09: // HT
		t.c = t.nextTab(t.c)
		t.wrapNext = false
	case 0x0a, 0x0b, 0x0c: // LF, VT, FF
		t.lineFeed()
	case 0x0d: // CR
		t.c, t.wrapNext = 0, false
	case 0x0e: // SO
		t.shift = true
	case 0x0f: // SI
		t.shift = false
	}
}

func (t *Terminal) nextTab(c int) int {
	for i := c + 1; i < t.cols; i++ {
		if t.tabs[i] {
			return i
		}
	}
	return t.cols - 1
}

func (t *Terminal) prevTab(c int) int {
	for i := c - 1; i > 0; i-- {
		if t.tabs[i] {
			return i
		}
	}
	return 0
}

// ground handles a printable byte, assembling UTF-8.
func (t *Terminal) ground(b byte) {
	if b < 0x80 {
		t.utf = nil
		t.print(rune(b))
		return
	}
	t.utf = append(t.utf, b)
	if !utf8.FullRune(t.utf) {
		if len(t.utf) >= utf8.UTFMax {
			t.utf = nil
			t.print(utf8.RuneError)
		}
		return
	}
	r, _ := utf8.DecodeRune(t.utf)
	t.utf = nil
	t.print(r)
}

func (t *Terminal) escape(b byte) {
	switch {
	case b == '[':
		t.state = stCSI
		return
	case b == ']':
		t.state, t.str, t.strOSC = stStr, t.str[:0], true
		return
	case b == 'P' || b == 'X' || b == '^' || b == '_':
		t.state, t.str, t.strOSC = stStr, t.str[:0], false
		return
	case b >= 0x20 && b <= 0x2f: // an intermediate: charset selection, DEC alignment
		t.inter = append(t.inter[:0], b)
		t.state = stEscInter
		return
	}
	t.state = stGround
	switch b {
	case '7':
		t.saveCursor()
	case '8':
		t.restoreCursor()
	case 'D':
		t.lineFeed()
	case 'E':
		t.c, t.wrapNext = 0, false
		t.lineFeed()
	case 'M':
		t.reverseIndex()
	case 'H':
		if t.c < len(t.tabs) {
			t.tabs[t.c] = true
		}
	case 'c':
		t.reset()
	case '=':
		t.appKeypad = true
	case '>':
		t.appKeypad = false
	}
}

func (t *Terminal) escInter(b byte) {
	t.state = stGround
	if b < 0x30 { // another intermediate; give up on the sequence
		return
	}
	switch t.inter[0] {
	case '(':
		t.g[0] = charset(b)
	case ')':
		t.g[1] = charset(b)
	}
}

// charset maps the final byte of a charset designation to the sets known here.
func charset(b byte) byte {
	if b == '0' {
		return '0'
	}
	return 'B'
}

func (t *Terminal) reset() {
	cols, rows := t.cols, t.rows
	back, keep := t.scrollback, t.maxBack
	reply := t.reply
	*t = *New(cols, rows)
	t.scrollback, t.maxBack, t.reply = back, keep, reply
}

func (t *Terminal) saveCursor() {
	t.saved.r, t.saved.c, t.saved.pen, t.saved.wrapNext = t.r, t.c, t.pen, t.wrapNext
	t.saved.g, t.saved.shift, t.saved.set = t.g, t.shift, true
}

func (t *Terminal) restoreCursor() {
	if !t.saved.set {
		t.r, t.c, t.wrapNext = 0, 0, false
		return
	}
	t.r, t.c = min(t.saved.r, t.rows-1), min(t.saved.c, t.cols-1)
	t.pen, t.wrapNext, t.g, t.shift = t.saved.pen, t.saved.wrapNext, t.saved.g, t.saved.shift
}

// feedString collects an OSC, DCS, APC, PM or SOS string.
func (t *Terminal) feedString(b byte) {
	switch {
	case b == 0x07:
		t.endString()
	case b == 0x1b:
		t.state = stStrEsc
	case len(t.str) < 4096: // a longer string is cut: only titles are read
		t.str = append(t.str, b)
	}
}

func (t *Terminal) endString() {
	t.state = stGround
	if t.strOSC {
		code, text, _ := strings.Cut(string(t.str), ";")
		if code == "0" || code == "2" { // icon name and title; the others (52, 8, 133, ...) are skipped
			t.title = cleanTitle(text)
		}
	}
	t.str = t.str[:0]
}

// cleanTitle keeps printable text of a title.
func cleanTitle(s string) string {
	s = ansi.Sanitize(strings.ToValidUTF8(s, ""))
	if len([]rune(s)) > 200 {
		s = string([]rune(s)[:200])
	}
	return s
}

// lineFeed moves down a row, scrolling the region at its bottom.
func (t *Terminal) lineFeed() {
	t.wrapNext = false
	switch {
	case t.r == t.bot:
		t.scrollUp(1)
	case t.r < t.rows-1:
		t.r++
	}
}

func (t *Terminal) reverseIndex() {
	t.wrapNext = false
	switch {
	case t.r == t.top:
		t.scrollDown(1)
	case t.r > 0:
		t.r--
	}
}

// scrollUp moves the rows of the region up n lines, blanking the bottom ones. On
// the main screen, a region that is the whole screen sends the lines that leave
// to the scrollback.
func (t *Terminal) scrollUp(n int) {
	g := t.cur().grid
	n = min(n, t.bot-t.top+1)
	if !t.useAlt && t.top == 0 {
		for _, row := range g[:n] {
			t.pushBack(row)
		}
	}
	copy(g[t.top:], g[t.top+n:t.bot+1])
	for i := t.bot - n + 1; i <= t.bot; i++ {
		g[i] = t.blankRow(t.pen.BG)
	}
}

func (t *Terminal) scrollDown(n int) {
	g := t.cur().grid
	n = min(n, t.bot-t.top+1)
	copy(g[t.top+n:t.bot+1], g[t.top:t.bot+1-n])
	for i := t.top; i < t.top+n; i++ {
		g[i] = t.blankRow(t.pen.BG)
	}
}

// print draws a character at the cursor.
func (t *Terminal) print(r rune) {
	if unsafeRune(r) {
		return
	}
	if t.g[b2i(t.shift)] == '0' {
		r = lineDrawing(r)
	}
	w := runeWidth(r)
	grid := t.cur().grid
	if w == 0 { // a combining mark joins the cluster before the cursor
		x := t.c
		if !t.wrapNext {
			x--
		}
		for x > 0 && grid[t.r][x].Width == 0 {
			x--
		}
		if x >= 0 && grid[t.r][x].Width > 0 {
			grid[t.r][x].Cluster += string(r)
			t.last = grid[t.r][x].Cluster
		}
		return
	}
	if t.wrapNext && t.autowrap {
		t.c, t.wrapNext = 0, false
		t.lineFeed()
		grid = t.cur().grid
	}
	if w == 2 && t.c == t.cols-1 { // a wide cluster does not fit in the last column
		if !t.autowrap {
			return
		}
		grid[t.r][t.c] = blankCell(t.pen.BG)
		t.c = 0
		t.lineFeed()
		grid = t.cur().grid
	}
	row := grid[t.r]
	if t.insertMode {
		copy(row[t.c+w:], row[t.c:])
	}
	t.clearWide(row, t.c)
	if w == 2 {
		t.clearWide(row, t.c+1)
	}
	cell := t.pen
	cell.Cluster, cell.Width = string(r), uint8(w) // #nosec G115 -- w is 1 or 2
	row[t.c] = cell
	if w == 2 {
		row[t.c+1] = Cell{Width: 0, Attrs: t.pen.Attrs, FG: t.pen.FG, BG: t.pen.BG}
	}
	t.last = cell.Cluster
	if t.c+w >= t.cols {
		t.c = t.cols - 1
		t.wrapNext = t.autowrap
	} else {
		t.c += w
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// clearWide blanks the other half of a wide cluster that the cell at x is about
// to overwrite.
func (t *Terminal) clearWide(row []Cell, x int) {
	switch {
	case row[x].Width == 2 && x+1 < len(row):
		row[x+1] = blankCell(t.pen.BG)
	case row[x].Width == 0 && x > 0:
		row[x-1] = blankCell(t.pen.BG)
	}
}

// runeWidth is the columns a character takes: 0, 1 or 2.
func runeWidth(r rune) int {
	if r < 0x20 || r == 0x7f {
		return 0
	}
	if r < 0x7f {
		return 1
	}
	return min(ansi.Width(string(r)), 2)
}

// lineDrawing maps ASCII to the DEC special graphics set.
func lineDrawing(r rune) rune {
	const set = "◆▒␉␌␍␊°±␤␋┘┐┌└┼⎺⎻─⎼⎽├┤┴┬│≤≥π≠£·"
	if r < '`' || r > '~' {
		return r
	}
	return []rune(set)[r-'`']
}

// unsafeRune reports a character that is dropped: the C1 controls, which a
// terminal may read as the start of a sequence, and the bidirectional
// overrides, which reorder text on the screen.
func unsafeRune(r rune) bool {
	return r >= 0x80 && r <= 0x9f || r >= 0x202a && r <= 0x202e || r >= 0x2066 && r <= 0x2069
}
