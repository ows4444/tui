package vt

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
)

// csiByte collects a CSI sequence and runs it at its final byte.
func (t *Terminal) csiByte(b byte) {
	switch {
	case b >= 0x30 && b <= 0x3f:
		if len(t.params) < 256 {
			t.params = append(t.params, b)
		}
	case b >= 0x20 && b <= 0x2f:
		if len(t.inter) < 4 {
			t.inter = append(t.inter, b)
		}
	case b >= 0x40 && b <= 0x7e:
		t.state = stGround
		t.csi(string(t.params), string(t.inter), b)
		t.params, t.inter = t.params[:0], t.inter[:0]
	default:
		t.state = stGround
	}
}

// nums parses the numbers of a parameter list; an empty one is 0.
func nums(params string) []int {
	params = strings.TrimLeft(params, "?<=>")
	if params == "" {
		return nil
	}
	parts := strings.Split(params, ";")
	out := make([]int, len(parts))
	for i, p := range parts {
		p, _, _ = strings.Cut(p, ":")
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			n = 0
		}
		out[i] = min(n, 1<<16)
	}
	return out
}

// csi runs a control sequence.
func (t *Terminal) csi(params, inter string, final byte) {
	if inter != "" {
		return // DECSCUSR and the like
	}
	prefix := byte(0)
	if params != "" && strings.IndexByte("?<=>", params[0]) >= 0 {
		prefix = params[0]
	}
	n := nums(params)
	arg := func(i, def int) int {
		if i < len(n) && n[i] > 0 {
			return n[i]
		}
		return def
	}
	if prefix == '?' {
		t.privateMode(n, final)
		return
	}
	if prefix != 0 {
		if final == 'c' && prefix == '>' {
			t.send("\x1b[>0;0;0c")
		}
		return
	}
	t.wrapNext = t.wrapNext && final == 'm' // every other sequence cancels a pending wrap
	switch final {
	case 'A':
		t.r = max(t.r-arg(0, 1), t.regionTop())
	case 'B', 'e':
		t.r = min(t.r+arg(0, 1), t.regionBottom())
	case 'C', 'a':
		t.c = min(t.c+arg(0, 1), t.cols-1)
	case 'D':
		t.c = max(t.c-arg(0, 1), 0)
	case 'E':
		t.r, t.c = min(t.r+arg(0, 1), t.regionBottom()), 0
	case 'F':
		t.r, t.c = max(t.r-arg(0, 1), t.regionTop()), 0
	case 'G', '`':
		t.c = min(arg(0, 1)-1, t.cols-1)
	case 'H', 'f':
		t.r, t.c = min(arg(0, 1)-1, t.rows-1), min(arg(1, 1)-1, t.cols-1)
	case 'd':
		t.r = min(arg(0, 1)-1, t.rows-1)
	case 'I':
		for i := arg(0, 1); i > 0; i-- {
			t.c = t.nextTab(t.c)
		}
	case 'Z':
		for i := arg(0, 1); i > 0; i-- {
			t.c = t.prevTab(t.c)
		}
	case 'J':
		t.eraseDisplay(arg0(n))
	case 'K':
		t.eraseLine(arg0(n))
	case 'L':
		t.insertLines(arg(0, 1))
	case 'M':
		t.deleteLines(arg(0, 1))
	case '@':
		t.insertChars(arg(0, 1))
	case 'P':
		t.deleteChars(arg(0, 1))
	case 'X':
		t.eraseChars(arg(0, 1))
	case 'S':
		t.scrollUp(arg(0, 1))
	case 'T':
		t.scrollDown(arg(0, 1))
	case 'b':
		for i := min(arg(0, 1), 1<<12); i > 0 && t.last != ""; i-- {
			t.printCluster(t.last)
		}
	case 'g':
		switch arg0(n) {
		case 0:
			t.tabs[t.c] = false
		case 3:
			clear(t.tabs)
		}
	case 'r':
		t.setRegion(arg(0, 1)-1, arg(1, t.rows)-1)
	case 's':
		t.saveCursor()
	case 'u':
		t.restoreCursor()
	case 'm':
		t.sgr(params)
	case 'h', 'l':
		if arg0(n) == 4 {
			t.insertMode = final == 'h'
		}
	case 'n':
		switch arg0(n) {
		case 5:
			t.send("\x1b[0n")
		case 6:
			t.send("\x1b[" + strconv.Itoa(t.r+1) + ";" + strconv.Itoa(t.c+1) + "R")
		}
	case 'c':
		if arg0(n) == 0 {
			t.send("\x1b[?62;22c")
		}
	}
}

func arg0(n []int) int {
	if len(n) > 0 {
		return n[0]
	}
	return 0
}

func (t *Terminal) regionTop() int {
	if t.r >= t.top {
		return t.top
	}
	return 0
}

func (t *Terminal) regionBottom() int {
	if t.r <= t.bot {
		return t.bot
	}
	return t.rows - 1
}

func (t *Terminal) setRegion(top, bot int) {
	if top < 0 || bot >= t.rows || top >= bot {
		top, bot = 0, t.rows-1
	}
	t.top, t.bot = top, bot
	t.r, t.c = 0, 0
}

// printCluster prints a cluster again, for REP.
func (t *Terminal) printCluster(s string) {
	for _, r := range s {
		t.print(r)
	}
}

func (t *Terminal) send(s string) {
	if t.reply != nil {
		t.reply([]byte(s))
	}
}

func (t *Terminal) eraseRange(r, from, to int) {
	row := t.cur().grid[r]
	to = min(to, t.cols)
	if from > 0 && from < t.cols && row[from].Width == 0 {
		row[from-1] = blankCell(t.pen.BG)
	}
	if to < t.cols && to > 0 && row[to].Width == 0 {
		row[to] = blankCell(t.pen.BG)
	}
	for c := max(from, 0); c < to; c++ {
		row[c] = blankCell(t.pen.BG)
	}
}

func (t *Terminal) eraseDisplay(mode int) {
	switch mode {
	case 0:
		t.eraseRange(t.r, t.c, t.cols)
		for r := t.r + 1; r < t.rows; r++ {
			t.eraseRange(r, 0, t.cols)
		}
	case 1:
		for r := 0; r < t.r; r++ {
			t.eraseRange(r, 0, t.cols)
		}
		t.eraseRange(t.r, 0, t.c+1)
	case 2:
		for r := 0; r < t.rows; r++ {
			t.eraseRange(r, 0, t.cols)
		}
	case 3:
		t.scrollback = nil
	}
}

func (t *Terminal) eraseLine(mode int) {
	switch mode {
	case 0:
		t.eraseRange(t.r, t.c, t.cols)
	case 1:
		t.eraseRange(t.r, 0, t.c+1)
	case 2:
		t.eraseRange(t.r, 0, t.cols)
	}
}

func (t *Terminal) insertLines(n int) {
	if t.r < t.top || t.r > t.bot {
		return
	}
	g := t.cur().grid
	n = min(n, t.bot-t.r+1)
	copy(g[t.r+n:t.bot+1], g[t.r:t.bot+1-n])
	for i := t.r; i < t.r+n; i++ {
		g[i] = t.blankRow(t.pen.BG)
	}
	t.c = 0
}

func (t *Terminal) deleteLines(n int) {
	if t.r < t.top || t.r > t.bot {
		return
	}
	g := t.cur().grid
	n = min(n, t.bot-t.r+1)
	copy(g[t.r:], g[t.r+n:t.bot+1])
	for i := t.bot - n + 1; i <= t.bot; i++ {
		g[i] = t.blankRow(t.pen.BG)
	}
	t.c = 0
}

func (t *Terminal) insertChars(n int) {
	row := t.cur().grid[t.r]
	n = min(n, t.cols-t.c)
	copy(row[t.c+n:], row[t.c:])
	for i := t.c; i < t.c+n; i++ {
		row[i] = blankCell(t.pen.BG)
	}
	t.fixWide(row)
}

func (t *Terminal) deleteChars(n int) {
	row := t.cur().grid[t.r]
	n = min(n, t.cols-t.c)
	copy(row[t.c:], row[t.c+n:])
	for i := t.cols - n; i < t.cols; i++ {
		row[i] = blankCell(t.pen.BG)
	}
	t.fixWide(row)
}

func (t *Terminal) eraseChars(n int) { t.eraseRange(t.r, t.c, t.c+n) }

// fixWide blanks halves of wide clusters that a shift of the row split.
func (t *Terminal) fixWide(row []Cell) {
	for i, c := range row {
		switch {
		case c.Width == 2 && (i+1 >= len(row) || row[i+1].Width != 0):
			row[i] = blankCell(c.BG)
		case c.Width == 0 && (i == 0 || row[i-1].Width != 2):
			row[i] = blankCell(c.BG)
		}
	}
}

// privateMode sets or resets the DEC private modes.
func (t *Terminal) privateMode(modes []int, final byte) {
	if final != 'h' && final != 'l' {
		return
	}
	on := final == 'h'
	for _, m := range modes {
		switch m {
		case 1:
			t.appCursor = on
		case 7:
			t.autowrap = on
		case 25:
			t.curHidden = !on
		case 47, 1047, 1049:
			t.setAlt(on, m == 1049)
		case 1000:
			t.setMouse(on, MouseClick)
		case 1002:
			t.setMouse(on, MouseDrag)
		case 1003:
			t.setMouse(on, MouseAny)
		case 1006:
			t.sgrMouse = on
		case 2004:
			t.bracketed = on
		}
	}
}

func (t *Terminal) setMouse(on bool, m MouseMode) {
	switch {
	case on:
		t.mouse = max(t.mouse, m)
	case t.mouse == m:
		t.mouse = MouseNone
	}
}

// setAlt switches screens. Entering the alternate screen with save (?1049)
// remembers the cursor and clears it.
func (t *Terminal) setAlt(on, save bool) {
	if on == t.useAlt {
		return
	}
	if on {
		if save {
			t.saveCursor()
		}
		t.useAlt = true
		t.alt.grid = t.blankGrid(t.rows)
		t.top, t.bot = 0, t.rows-1
		return
	}
	t.useAlt = false
	t.top, t.bot = 0, t.rows-1
	if save {
		t.restoreCursor()
	}
}

// sgr applies a Select Graphic Rendition parameter list to the pen.
func (t *Terminal) sgr(params string) {
	if params == "" {
		t.pen = Cell{}
		return
	}
	parts := strings.Split(params, ";")
	for i := 0; i < len(parts); i++ {
		sub := strings.Split(parts[i], ":")
		n, err := strconv.Atoi(sub[0])
		if sub[0] == "" {
			n, err = 0, nil
		}
		if err != nil {
			continue
		}
		switch {
		case n == 0:
			t.pen = Cell{}
		case n == 1:
			t.pen.Attrs |= Bold
		case n == 2:
			t.pen.Attrs |= Dim
		case n == 3:
			t.pen.Attrs |= Italic
		case n == 4:
			t.pen.Attrs |= Underline
		case n == 5 || n == 6:
			t.pen.Attrs |= Blink
		case n == 7:
			t.pen.Attrs |= Reverse
		case n == 8:
			t.pen.Attrs |= Conceal
		case n == 9:
			t.pen.Attrs |= Strike
		case n == 22:
			t.pen.Attrs &^= Bold | Dim
		case n == 23:
			t.pen.Attrs &^= Italic
		case n == 24:
			t.pen.Attrs &^= Underline
		case n == 25:
			t.pen.Attrs &^= Blink
		case n == 27:
			t.pen.Attrs &^= Reverse
		case n == 28:
			t.pen.Attrs &^= Conceal
		case n == 29:
			t.pen.Attrs &^= Strike
		case n >= 30 && n <= 37:
			t.pen.FG = ansi.BasicColor(n - 30)
		case n == 39:
			t.pen.FG = nil
		case n >= 40 && n <= 47:
			t.pen.BG = ansi.BasicColor(n - 40)
		case n == 49:
			t.pen.BG = nil
		case n >= 90 && n <= 97:
			t.pen.FG = ansi.BasicColor(n - 90 + 8)
		case n >= 100 && n <= 107:
			t.pen.BG = ansi.BasicColor(n - 100 + 8)
		case n == 38 || n == 48 || n == 58: // 58 is the underline colour, which cells do not keep
			c, used := extColor(sub, parts[i+1:])
			i += used
			switch {
			case c == nil || n == 58:
			case n == 38:
				t.pen.FG = c
			default:
				t.pen.BG = c
			}
		}
	}
}

// extColor reads the colour after a 38 or 48: "5;n" or "2;r;g;b" in the
// separate-parameter form, or the colon forms in sub. used is how many of the
// following parameters it took.
func extColor(sub, rest []string) (c ansi.Color, used int) {
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return min(max(n, 0), 255) }
	if len(sub) > 1 { // 38:5:n, 38:2:r:g:b, 38:2::r:g:b
		switch sub[1] {
		case "5":
			if len(sub) > 2 {
				return ansi.Color256(atoi(sub[2])), 0 // #nosec G115 -- atoi clamps to 0..255
			}
		case "2":
			v := sub[2:]
			if len(v) == 4 { // the colour-space id is empty or a number
				v = v[1:]
			}
			if len(v) >= 3 {
				return ansi.RGB{R: uint8(atoi(v[0])), G: uint8(atoi(v[1])), B: uint8(atoi(v[2]))}, 0 // #nosec G115 -- clamped to 255
			}
		}
		return nil, 0
	}
	if len(rest) >= 2 && rest[0] == "5" {
		return ansi.Color256(atoi(rest[1])), 2 // #nosec G115 -- atoi clamps to 0..255
	}
	if len(rest) >= 4 && rest[0] == "2" {
		return ansi.RGB{R: uint8(atoi(rest[1])), G: uint8(atoi(rest[2])), B: uint8(atoi(rest[3]))}, 4 // #nosec G115 -- clamped to 255
	}
	return nil, 0
}
