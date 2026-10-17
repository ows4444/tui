// Command hexedit edits a file as bytes: the offset, the bytes in hex and the
// same bytes as text, side by side, with a cursor, a selection, search, go to
// offset, insert and delete, undo, and a small inspector that reads the bytes
// under the cursor as numbers. With no file it opens a sample.
//
// Mouse: click a byte in either pane to put the cursor there, drag to select,
// scroll the wheel. Keyboard: arrows move (Shift extends the selection), PgUp
// and PgDn page, Home and End go to the start and end of the row, Ctrl+Home and
// Ctrl+End to the first and last byte; Tab swaps between the hex and the text
// pane; type hex digits (hex pane) or characters (text pane) to overwrite;
// Ctrl+F and Ctrl+G find and go to from either pane (/ and : work in the hex pane);
// Insert switches to inserting; Delete and Backspace remove bytes; / searches
// (hex like "de ad", or text in quotes), n and N go to the next and previous
// match; : goes to an offset; y copies the selection as hex, Y as text; Ctrl+Z
// and Ctrl+Y undo and redo; Ctrl+S saves; F1 shows help; Ctrl+Q quits (asking
// first if changes would be lost), Ctrl+C quits at once.
//
//	go run ./examples/hexedit [file]
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	chrome    = 5 // title, column header, status, message, hints
	inspectAt = 104
)

type pane int

const (
	inHex pane = iota
	inText
)

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeGoto
	modeSaveAs
	modeDiscard
)

type model struct {
	st    store
	path  string
	buf   buffer
	saved string // a fingerprint of the bytes as last saved, to tell if they changed

	cur    int // cursor byte; len(data) is the place after the last byte
	nib    int // 0 for the high digit, 1 for the low, in the hex pane
	anchor int // other end of the selection, or -1
	pane   pane
	insert bool
	top    int // first row shown

	mode mode
	in   textinput.Model
	last []byte // the last pattern searched for
	msg  string
	help bool
	drag bool

	width, height int
}

func newModel(st store, path string) (model, error) {
	data := sampleBytes()
	if path != "" {
		b, err := st.read(path)
		if err != nil {
			return model{}, err
		}
		data = b
	}
	in := textinput.New()
	in.Width = 40
	m := model{st: st, path: path, buf: buffer{data: data}, anchor: -1, in: in}
	m.saved = fingerprint(data)
	return m, nil
}

func fingerprint(b []byte) string {
	// Length and a cheap rolling sum: enough to notice an edit, and an edit
	// that restores the original bytes counts as unchanged.
	var h uint64 = 1469598103934665603
	for _, x := range b {
		h = (h ^ uint64(x)) * 1099511628211
	}
	return fmt.Sprintf("%d:%x", len(b), h)
}

func (m model) Init() tui.Cmd { return nil }

func (m model) dirty() bool { return fingerprint(m.buf.data) != m.saved }

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// perRow is how many bytes a row shows: 16 if they fit, else 8, else 4.
func (m model) perRow() int {
	switch w := m.size().W; {
	case w >= 77:
		return 16
	case w >= 44:
		return 8
	}
	return 4
}

func (m model) bodyH() int { return max(1, m.size().H-chrome) }

func (m model) rows() int { return len(m.buf.data)/m.perRow() + 1 }

const offW = 8

// hexX is the column where byte i of a row starts in the hex pane.
func (m model) hexX(i int) int {
	x := offW + 2 + i*3
	if m.perRow() > 8 && i >= 8 {
		x++
	}
	return x
}

func (m model) textX() int { return m.hexX(m.perRow()-1) + 4 }

func (m model) end() int { return len(m.buf.data) }

// clamp keeps the cursor inside the file (or just past its end) and scrolls to
// show it.
func (m model) clamp() model {
	m.cur = max(0, min(m.cur, m.end()))
	if m.anchor > m.end() {
		m.anchor = m.end()
	}
	row := m.cur / m.perRow()
	if row < m.top {
		m.top = row
	}
	if row >= m.top+m.bodyH() {
		m.top = row - m.bodyH() + 1
	}
	m.top = max(0, min(m.top, max(0, m.rows()-m.bodyH())))
	return m
}

func (m model) selection() (lo, hi int, ok bool) {
	if m.anchor < 0 || m.anchor == m.cur {
		return 0, 0, false
	}
	return min(m.anchor, m.cur), min(max(m.anchor, m.cur), m.end()-1), true
}

// ---- editing ---------------------------------------------------------------

func (m model) nibble(v byte) model {
	m.anchor = -1
	d := m.buf.data
	appending := m.insert || m.cur >= len(d)
	switch {
	case m.nib == 0 && appending:
		m.buf.replace(m.cur, 0, []byte{v << 4}, false)
		m.nib = 1
	case m.nib == 0:
		m.buf.replace(m.cur, 1, []byte{v<<4 | d[m.cur]&0x0f}, false)
		m.nib = 1
	default:
		m.buf.replace(m.cur, 1, []byte{d[m.cur]&0xf0 | v}, true)
		m.nib = 0
		m.cur++
	}
	m.msg = ""
	return m.clamp()
}

func (m model) typeChar(c byte) model {
	m.anchor = -1
	if m.insert || m.cur >= m.end() {
		m.buf.replace(m.cur, 0, []byte{c}, false)
	} else {
		m.buf.replace(m.cur, 1, []byte{c}, false)
	}
	m.cur++
	m.nib = 0
	m.msg = ""
	return m.clamp()
}

func (m model) remove(back bool) model {
	if lo, hi, ok := m.selection(); ok {
		m.buf.replace(lo, hi-lo+1, nil, false)
		m.cur, m.anchor, m.nib = lo, -1, 0
		return m.clamp()
	}
	at := m.cur
	if back {
		at--
	}
	if at < 0 || at >= m.end() {
		return m
	}
	m.buf.replace(at, 1, nil, false)
	m.cur, m.nib = at, 0
	return m.clamp()
}

func (m model) save(path string) model {
	if err := m.st.write(path, m.buf.data); err != nil {
		m.msg = "Cannot save: " + err.Error()
		return m
	}
	m.path, m.saved = path, fingerprint(m.buf.data)
	m.msg = fmt.Sprintf("Saved %s (%d bytes)", path, len(m.buf.data))
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.clamp(), nil
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
	case modeSearch, modeGoto, modeSaveAs:
		return m.promptKey(k)
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	if k.Type == tui.KeyCtrl {
		return m.ctrlKey(k.Code)
	}
	if next, ok := m.moveKey(k); ok {
		return next.clamp(), nil
	}
	return m.editKey(k)
}

func (m model) ctrlKey(code rune) (tui.Model, tui.Cmd) {
	switch code {
	case 's':
		if m.path == "" {
			return m.startPrompt(modeSaveAs, "Save as ▸ ", ""), nil
		}
		return m.save(m.path), nil
	case 'z':
		if at, ok := m.buf.undoStep(); ok {
			m.cur, m.nib, m.anchor, m.msg = at, 0, -1, "Undone"
		} else {
			m.msg = "Nothing to undo"
		}
		return m.clamp(), nil
	case 'y':
		if at, ok := m.buf.redoStep(); ok {
			m.cur, m.nib, m.anchor, m.msg = at, 0, -1, "Redone"
		} else {
			m.msg = "Nothing to redo"
		}
		return m.clamp(), nil
	case 'a':
		m.anchor, m.cur = 0, max(0, m.end()-1)
		return m, nil
	case 'f':
		return m.startPrompt(modeSearch, "Find ▸ ", ""), nil
	case 'g':
		return m.startPrompt(modeGoto, "Go to ▸ ", ""), nil
	case 'q':
		if m.dirty() {
			m.mode = modeDiscard
			return m, nil
		}
		return m, tui.Quit()
	}
	return m, nil
}

func (m model) moveKey(k tui.Key) (model, bool) {
	n := m.perRow()
	shift := k.Mod.Shift()
	from := m.cur
	switch k.Type {
	case tui.KeyLeft:
		m.cur--
	case tui.KeyRight:
		m.cur++
	case tui.KeyUp:
		m.cur -= n
	case tui.KeyDown:
		m.cur += n
	case tui.KeyPgUp:
		m.cur -= n * max(1, m.bodyH()-1)
	case tui.KeyPgDown:
		m.cur += n * max(1, m.bodyH()-1)
	case tui.KeyHome:
		m.cur -= m.cur % n
	case tui.KeyEnd:
		m.cur = min(m.end(), m.cur-m.cur%n+n-1)
	case tui.KeyTab:
		m.pane = 1 - m.pane
		m.nib = 0
		return m, true
	default:
		return m, false
	}
	m.cur = max(0, min(m.cur, m.end()))
	switch {
	case shift && m.anchor < 0:
		m.anchor = from
	case !shift:
		m.anchor = -1
	}
	m.nib = 0
	m.msg = ""
	return m, true
}

func (m model) editKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.anchor, m.nib = -1, 0
		return m, nil
	case tui.KeyDelete:
		return m.remove(false), nil
	case tui.KeyBackspace:
		return m.remove(true), nil
	case tui.KeyInsert:
		m.insert = !m.insert
		return m, nil
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) runeKey(s string) (tui.Model, tui.Cmd) {
	r := []rune(s)
	if len(r) != 1 {
		return m, nil
	}
	c := r[0]
	if m.pane == inText && c >= 0x20 && c < 0x7f {
		return m.typeChar(byte(c)), nil // in the text pane every printable key is data
	}
	if m.pane == inHex {
		if v, ok := hexDigit(c); ok {
			return m.nibble(v), nil
		}
	}
	return m.command(c)
}

func hexDigit(c rune) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return byte(c - '0'), true
	case c >= 'a' && c <= 'f':
		return byte(c-'a') + 10, true
	case c >= 'A' && c <= 'F':
		return byte(c-'A') + 10, true
	}
	return 0, false
}

func (m model) command(c rune) (tui.Model, tui.Cmd) {
	switch c {
	case '/':
		return m.startPrompt(modeSearch, "Find ▸ ", ""), nil
	case ':':
		return m.startPrompt(modeGoto, "Go to ▸ ", ""), nil
	case 'n':
		return m.findNext(false), nil
	case 'N':
		return m.findNext(true), nil
	case 'y':
		return m.copySelection(false)
	case 'Y':
		return m.copySelection(true)
	case 'u':
		return m.ctrlKey('z')
	case 'q':
		return m.ctrlKey('q')
	case 'x':
		return m.remove(false), nil
	}
	return m, nil
}

func (m model) copySelection(asText bool) (tui.Model, tui.Cmd) {
	data := m.buf.data
	lo, hi := m.cur, m.cur
	if l, h, ok := m.selection(); ok {
		lo, hi = l, h
	}
	if lo >= len(data) {
		m.msg = "Nothing to copy"
		return m, nil
	}
	b := data[lo : hi+1]
	text := hexString(b)
	if asText {
		text = string(b)
	}
	m.msg = fmt.Sprintf("Copied %d bytes (if your terminal allows it)", len(b))
	return m, tui.WriteClipboard(text)
}

func (m model) findNext(back bool) model {
	if len(m.last) == 0 {
		m.msg = "Search first with /"
		return m
	}
	base := m.cur
	if lo, hi, ok := m.selection(); ok && hi-lo+1 == len(m.last) {
		base = lo // the match just found is selected: go on from its start
	}
	from := base + 1
	if back {
		from = base
	}
	at := find(m.buf.data, m.last, from, back)
	if at < 0 {
		m.msg = "Not found: " + hexString(m.last)
		return m
	}
	m.cur, m.anchor, m.nib = at, -1, 0
	if len(m.last) > 1 {
		m.anchor, m.cur = at, at+len(m.last)-1
	}
	m.msg = fmt.Sprintf("Found at 0x%x", at)
	return m.clamp()
}

// ---- prompts ---------------------------------------------------------------

func (m model) startPrompt(md mode, label, initial string) model {
	m.mode = md
	m.in.Prompt = label
	m.in.SetValue(initial)
	m.in.CursorEnd()
	m.in.Focus()
	return m
}

func (m model) promptKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.mode = modeNormal
		m.in.Blur()
		return m, nil
	case tui.KeyEnter:
		return m.submit(), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

func (m model) submit() model {
	v := m.in.Value()
	md := m.mode
	m.mode = modeNormal
	m.in.Blur()
	switch md {
	case modeSearch:
		pat, err := parsePattern(v)
		if err != nil {
			m.msg = sentence(err)
			return m
		}
		m.last = pat
		m.cur--
		return m.findNext(false)
	case modeGoto:
		at, err := parseOffset(v, m.cur, m.end())
		if err != nil {
			m.msg = sentence(err)
			return m
		}
		m.cur, m.anchor, m.nib = at, -1, 0
		m.msg = fmt.Sprintf("Offset 0x%x", at)
		return m.clamp()
	case modeSaveAs:
		if p := strings.TrimSpace(v); p != "" {
			return m.save(p)
		}
	}
	return m
}

// sentence is an error's text as a message: it starts with a capital.
func sentence(err error) string {
	s := err.Error()
	return strings.ToUpper(s[:1]) + s[1:]
}

func (m model) discardKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && (k.Text == "y" || k.Text == "Y"):
		return m, tui.Quit()
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && (k.Text == "n" || k.Text == "N"):
		m.mode = modeNormal
	}
	return m, nil
}

// ---- mouse -----------------------------------------------------------------

// byteAt is the byte drawn at screen cell x, y and the pane it is in, or -1.
func (m model) byteAt(x, y int) (int, pane) {
	row := y - 2
	if row < 0 || row >= m.bodyH() {
		return -1, 0
	}
	base := (m.top + row) * m.perRow()
	n := m.perRow()
	for i := 0; i < n; i++ {
		if x >= m.hexX(i) && x <= m.hexX(i)+2 {
			return base + i, inHex
		}
	}
	if x >= m.textX() && x < m.textX()+n {
		return base + x - m.textX(), inText
	}
	return -1, 0
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.mode != modeNormal || m.help {
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.top = max(0, m.top-3)
		return m, nil
	case tui.MouseButtonWheelDown:
		m.top = min(max(0, m.rows()-m.bodyH()), m.top+3)
		return m, nil
	}
	if ev.Action == tui.MouseActionRelease {
		m.drag = false
		return m, nil
	}
	at, p := m.byteAt(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion && m.drag:
		if at >= 0 {
			m.cur = min(at, max(0, m.end()-1))
		}
	case ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress && at >= 0:
		m.pane, m.cur, m.nib = p, min(at, m.end()), 0
		m.anchor, m.drag = m.cur, at < m.end() // a drag from here makes a selection
	default:
		return m, nil
	}
	top := m.top
	m = m.clamp()
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

func (m model) header(th theme.Theme) string {
	n := m.perRow()
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", offW+2))
	for i := 0; i < n; i++ {
		if i == 8 && n > 8 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%02x ", i)
	}
	b.WriteString(" " + strings.Repeat(" ", 0))
	return ansi.NewStyle().Faint().Render(strings.TrimRight(b.String(), " "))
}

// cell is one byte's text styled for its state.
func (m model) cell(i int, text string, inThisPane bool, th theme.Theme) string {
	st := ansi.NewStyle()
	if i < m.end() {
		switch b := m.buf.data[i]; {
		case b == 0:
			st = ansi.NewStyle().Faint()
		case !printable(b):
			st = ansi.NewStyle().Foreground(th.Info)
		}
	}
	if lo, hi, ok := m.selection(); ok && i >= lo && i <= hi {
		st = st.Background(th.Selection).Bold()
	}
	if i == m.cur {
		if inThisPane {
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		} else {
			st = st.Underline().Bold()
		}
	}
	return st.Render(text)
}

func (m model) rowText(r int, th theme.Theme) string {
	n := m.perRow()
	base := r * n
	if base > m.end() {
		return ""
	}
	var hx, tx strings.Builder
	for i := 0; i < n; i++ {
		at := base + i
		if i == 8 && n > 8 {
			hx.WriteString(" ")
		}
		switch {
		case at < m.end():
			h := fmt.Sprintf("%02x", m.buf.data[at])
			if at == m.cur && m.pane == inHex && m.nib == 1 {
				h = m.cell(at, h[:1], false, th) + m.cell(at, h[1:], true, th)
			} else {
				h = m.cell(at, h, m.pane == inHex, th)
			}
			hx.WriteString(h + " ")
			tx.WriteString(m.cell(at, asciiOf(m.buf.data[at]), m.pane == inText, th))
		case at == m.end() && at == m.cur:
			hx.WriteString(m.cell(at, "  ", m.pane == inHex, th) + " ")
			tx.WriteString(m.cell(at, " ", m.pane == inText, th))
		default:
			hx.WriteString("   ")
		}
	}
	off := ansi.NewStyle().Faint().Render(fmt.Sprintf("%0*x", offW, base))
	return off + "  " + hx.String() + " " + tx.String()
}

func (m model) inspectorLines(th theme.Theme) []string {
	lines := inspect(m.buf.data, m.cur)
	if lo, hi, ok := m.selection(); ok {
		lines = append([]string{fmt.Sprintf("selected %d bytes", hi-lo+1)}, lines...)
	}
	for i, l := range lines {
		if k, v, ok := strings.Cut(l, "  "); ok {
			lines[i] = ansi.NewStyle().Faint().Render(k) + " " + strings.TrimSpace(v)
		}
	}
	return lines
}

func (m model) body(th theme.Theme) []layout.FlexChild {
	h := m.bodyH()
	side := m.size().W >= inspectAt
	insp := m.inspectorLines(th)
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	rowW := m.textX() + m.perRow()
	var out []layout.FlexChild
	for i := 0; i < h; i++ {
		line := pad(m.rowText(m.top+i, th), rowW)
		if side {
			extra := ""
			if i < len(insp) {
				extra = insp[i]
			}
			line += " " + rule + " " + extra
		}
		out = append(out, layout.FlexChild{Node: layout.Block(ansi.Truncate(line, m.size().W))})
	}
	return out
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	name := m.path
	if name == "" {
		name = "sample"
	}
	if m.dirty() {
		name += " ●"
	}
	mode := "OVR"
	if m.insert {
		mode = "INS"
	}
	where := "hex"
	if m.pane == inText {
		where = "text"
	}
	val := "end"
	if m.cur < m.end() {
		val = fmt.Sprintf("%02x", m.buf.data[m.cur])
	}
	left := fmt.Sprintf("%s  %d bytes  0x%x = %s  %s %s", name, m.end(), m.cur, val, where, mode)
	if lo, hi, ok := m.selection(); ok {
		left += fmt.Sprintf("  sel %d", hi-lo+1)
	}
	return ansi.NewStyle().Background(th.Surface).Render(pad(ansi.Truncate(left, w), w))
}

func (m model) messageLine(th theme.Theme) string {
	w := m.size().W
	switch m.mode {
	case modeDiscard:
		return ansi.NewStyle().Bold().Foreground(th.Warning).Render("Discard unsaved changes? ") + "[y] yes  [n] no"
	case modeSearch, modeGoto, modeSaveAs:
		return ansi.Truncate(m.in.View(), w)
	}
	return ansi.Truncate(m.msg, w)
}

func (m model) helpLines(th theme.Theme) []layout.FlexChild {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))}
	}
	return []layout.FlexChild{
		key("arrows", "move (Shift extends the selection)"), key("Tab", "hex pane or text pane"),
		key("0-9 a-f", "overwrite a hex digit"), key("text keys", "overwrite a byte (text pane)"),
		key("Insert", "insert instead of overwrite"), key("Delete Bksp", "remove bytes"),
		key("/ n N", "find hex (de ad) or \"text\", next, previous"), key(":", "go to hex offset, #decimal, +n, end"),
		key("y Y", "copy as hex, as text"), key("Ctrl+Z Ctrl+Y", "undo, redo"), key("Ctrl+S", "save"),
		key("Ctrl+Q", "quit"), key("F1", "close help"),
	}
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("hexedit")
	body := m.body(th)
	if m.help {
		body = m.helpLines(th)
		for len(body) < m.bodyH() {
			body = append(body, layout.FlexChild{Node: layout.Block("")})
		}
		body = body[:m.bodyH()]
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "tab", Action: "pane"}, widgets.Hint{Key: "/", Action: "find"},
		widgets.Hint{Key: ":", Action: "goto"}, widgets.Hint{Key: "ins", Action: "insert"}, widgets.Hint{Key: "ctrl+z", Action: "undo"},
		widgets.Hint{Key: "ctrl+s", Action: "save"}, widgets.Hint{Key: "f1", Action: "help"})
	rows := []layout.FlexChild{{Node: layout.Block(ansi.Truncate(title, w))}, {Node: layout.Block(ansi.Truncate(m.header(th), w))}}
	rows = append(rows, body...)
	rows = append(rows,
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(m.messageLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
	return layout.Column(0, rows...)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	m, err := newModel(diskStore{}, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hexedit:", err)
		os.Exit(1)
	}
	if _, err := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
