package vtscreen

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// Cell is one screen cell: its character and the SGR attributes in force when
// it was written. FG and BG are nil for the terminal's default colour.
type Cell struct {
	Rune                                          rune
	Bold, Dim, Italic, Underline, Reverse, Strike bool
	FG, BG                                        ansi.Color
}

// Screen is a minimal terminal emulator: a fixed cols x rows grid that
// interprets the small subset of ANSI the tui renderer emits (cursor
// movement, line/screen erase, "\r", "\n", autowrap) and nothing more.
// Unlike a byte-stream replay it models the physical screen, so it
// reproduces what a real terminal does when output wraps at the right edge,
// scrolls off the bottom, or the window is resized. It also tracks SGR attributes per cell and the cursor's
// visibility (DECTCEM). Other sequences are ignored; control strings (OSC,
// DCS, APC, PM, SOS) are skipped whole, so a hyperlink's URL never lands in the grid. Not safe for concurrent use.
type Screen struct {
	cols, rows int
	grid       [][]Cell
	pen        Cell // attributes applied to the next glyph (Rune unused)
	curHidden  bool
	r, c       int
	wrapNext   bool   // cursor is past the last column; next glyph wraps
	soft       []bool // soft[i]: row i wrapped into row i+1 (autowrap, not a newline)
	rest       []byte // incomplete trailing escape sequence / rune
}

// NewScreen returns a blank screen with the cursor at the top-left.
func NewScreen(cols, rows int) *Screen {
	s := &Screen{cols: cols, rows: rows}
	s.grid = s.blank(rows)
	s.soft = make([]bool, rows)
	return s
}

func (s *Screen) blank(n int) [][]Cell {
	g := make([][]Cell, n)
	for i := range g {
		g[i] = s.blankRow()
	}
	return g
}

func (s *Screen) blankRow() []Cell {
	row := make([]Cell, s.cols)
	for i := range row {
		row[i] = Cell{Rune: ' '}
	}
	return row
}

// Resize changes the grid size the way a terminal in the alternate screen
// does: content is cropped or extended (no reflow), the cursor is clamped.
func (s *Screen) Resize(cols, rows int) {
	g := make([][]Cell, rows)
	for i := range g {
		g[i] = make([]Cell, cols)
		for j := range g[i] {
			g[i][j] = Cell{Rune: ' '}
			if i < s.rows && j < s.cols {
				g[i][j] = s.grid[i][j]
			}
		}
	}
	s.cols, s.rows, s.grid = cols, rows, g
	s.soft = make([]bool, rows)
	s.r = min(s.r, rows-1)
	s.c = min(s.c, cols-1)
	s.wrapNext = false
}

// ResizeReflow changes the grid size the way a main-screen terminal does:
// rows joined by autowrap are re-wrapped to the new width, so a line drawn
// 100 columns wide occupies two rows at width 50 and the cursor moves with
// its text. Rows that no longer fit scroll off the top (the scrollback is
// not kept); blank rows below the cursor are dropped first.
func (s *Screen) ResizeReflow(cols, rows int) {
	if cols < 1 || rows < 1 {
		s.Resize(cols, rows)
		return
	}
	type logical struct{ cells []Cell }
	var lines []logical
	curLine, curOff := 0, 0
	var cur []Cell
	for i := 0; i < s.rows; i++ {
		if i == s.r {
			curLine, curOff = len(lines), len(cur)+s.c
		}
		cur = append(cur, s.grid[i]...)
		if s.soft[i] {
			continue
		}
		n := len(cur)
		for n > 0 && cur[n-1] == (Cell{Rune: ' '}) && !(i >= s.r && len(lines) == curLine && n <= curOff) {
			n--
		}
		lines = append(lines, logical{cur[:n]})
		cur = nil
	}
	if len(cur) > 0 {
		lines = append(lines, logical{cur})
	}
	var grid [][]Cell
	var soft []bool
	curRow, curCol := 0, 0
	for li, l := range lines {
		n := max(1, (len(l.cells)+cols-1)/cols)
		if li == curLine {
			curRow, curCol = len(grid)+curOff/cols, curOff%cols
			n = max(n, curOff/cols+1)
		}
		for k := 0; k < n; k++ {
			row := make([]Cell, cols)
			for j := range row {
				row[j] = Cell{Rune: ' '}
				if idx := k*cols + j; idx < len(l.cells) {
					row[j] = l.cells[idx]
				}
			}
			grid = append(grid, row)
			soft = append(soft, k < n-1)
		}
	}
	for len(grid) > rows && len(grid) > curRow+1 {
		grid, soft = grid[:len(grid)-1], soft[:len(soft)-1]
	}
	if drop := len(grid) - rows; drop > 0 {
		grid, soft, curRow = grid[drop:], soft[drop:], curRow-drop
	}
	for len(grid) < rows {
		row := make([]Cell, cols)
		for j := range row {
			row[j] = Cell{Rune: ' '}
		}
		grid, soft = append(grid, row), append(soft, false)
	}
	s.cols, s.rows, s.grid, s.soft = cols, rows, grid, soft
	s.r, s.c = min(max(curRow, 0), rows-1), min(curCol, cols-1)
	s.wrapNext = false
}

// Lines returns one string per screen row, right-trimmed.
func (s *Screen) Lines() []string {
	out := make([]string, s.rows)
	for i, row := range s.grid {
		rs := make([]rune, len(row))
		for j, c := range row {
			rs[j] = c.Rune
		}
		out[i] = strings.TrimRight(string(rs), " ")
	}
	return out
}

// Cell returns the cell at column x, row y; a blank default cell when the
// position is off the screen.
func (s *Screen) Cell(x, y int) Cell {
	if y < 0 || y >= s.rows || x < 0 || x >= s.cols {
		return Cell{Rune: ' '}
	}
	return s.grid[y][x]
}

// Cursor returns the cursor's column and row and whether it is visible.
func (s *Screen) Cursor() (x, y int, visible bool) { return s.c, s.r, !s.curHidden }

// Write feeds program output to the screen.
func (s *Screen) Write(p []byte) {
	b := append(s.rest, p...)
	s.rest = nil
	for i := 0; i < len(b); {
		switch ch := b[i]; {
		case ch == 0x1b:
			if i+1 >= len(b) {
				s.rest = append([]byte(nil), b[i:]...)
				return
			}
			switch b[i+1] {
			case '[':
			case ']', 'P', 'X', '^', '_': // OSC, DCS, SOS, PM, APC: a string up to BEL or ST
				end, ok := stringEnd(b, i+2)
				if !ok {
					s.rest = append([]byte(nil), b[i:]...)
					return
				}
				i = end
				continue
			default:
				i += 2
				continue
			}
			j := i + 2
			for j < len(b) && (b[j] < 0x40 || b[j] > 0x7e) {
				j++
			}
			if j >= len(b) {
				s.rest = append([]byte(nil), b[i:]...)
				return
			}
			s.csi(string(b[i+2:j]), b[j])
			i = j + 1
		case ch == '\r':
			s.c, s.wrapNext = 0, false
			i++
		case ch == '\n':
			s.lineFeed()
			i++
		case ch < 0x20:
			i++
		default:
			if !utf8.FullRune(b[i:]) {
				s.rest = append([]byte(nil), b[i:]...)
				return
			}
			r, n := utf8.DecodeRune(b[i:])
			s.put(r)
			i += n
		}
	}
}

// stringEnd returns the index just past the BEL or ST (ESC \) that ends the
// control string whose payload starts at from. ok is false when it is not
// complete yet.
func stringEnd(b []byte, from int) (end int, ok bool) {
	for j := from; j < len(b); j++ {
		switch {
		case b[j] == 0x07:
			return j + 1, true
		case b[j] == 0x1b && j+1 < len(b) && b[j+1] == '\\':
			return j + 2, true
		case b[j] == 0x1b && j+1 >= len(b):
			return 0, false
		}
	}
	return 0, false
}

func (s *Screen) lineFeed() {
	if s.r == s.rows-1 {
		copy(s.grid, s.grid[1:])
		copy(s.soft, s.soft[1:])
		s.grid[s.rows-1] = s.blankRow()
		s.soft[s.rows-1] = false
		return
	}
	s.r++
}

func (s *Screen) put(r rune) {
	if s.cols == 0 || s.rows == 0 {
		return
	}
	if s.wrapNext {
		s.soft[s.r] = true
		s.c, s.wrapNext = 0, false
		s.lineFeed()
	}
	cell := s.pen
	cell.Rune = r
	s.grid[s.r][s.c] = cell
	if s.c == s.cols-1 {
		s.wrapNext = true
	} else {
		s.c++
	}
}

func (s *Screen) csi(params string, final byte) {
	if strings.HasPrefix(params, "?") {
		// Private mode toggles: only cursor visibility matters here; the
		// alt screen, mouse and paste modes change nothing on this grid.
		if params == "?25" {
			switch final {
			case 'h':
				s.curHidden = false
			case 'l':
				s.curHidden = true
			}
		}
		return
	}
	arg := func(i, def int) int {
		parts := strings.Split(params, ";")
		if i < len(parts) {
			if n, err := strconv.Atoi(parts[i]); err == nil {
				return n
			}
		}
		return def
	}
	switch final {
	case 'A':
		s.r = max(0, s.r-max(1, arg(0, 1)))
		s.wrapNext = false
	case 'B':
		s.r = min(s.rows-1, s.r+max(1, arg(0, 1)))
		s.wrapNext = false
	case 'C':
		s.c = min(s.cols-1, s.c+max(1, arg(0, 1)))
		s.wrapNext = false
	case 'D':
		s.c = max(0, s.c-max(1, arg(0, 1)))
		s.wrapNext = false
	case 'G':
		s.c = min(s.cols-1, max(0, arg(0, 1)-1))
		s.wrapNext = false
	case 'H':
		s.r = min(s.rows-1, max(0, arg(0, 1)-1))
		s.c = min(s.cols-1, max(0, arg(1, 1)-1))
		s.wrapNext = false
	case 'm':
		s.sgr(params)
	case 'J':
		switch arg(0, 0) {
		case 0:
			s.eraseRow(s.r, s.c, s.cols)
			for r := s.r + 1; r < s.rows; r++ {
				s.eraseRow(r, 0, s.cols)
			}
		case 2:
			for r := 0; r < s.rows; r++ {
				s.eraseRow(r, 0, s.cols)
			}
		}
	case 'K':
		switch arg(0, 0) {
		case 0:
			s.eraseRow(s.r, s.c, s.cols)
		case 2:
			s.eraseRow(s.r, 0, s.cols)
		}
	}
}

func (s *Screen) eraseRow(r, from, to int) {
	if from == 0 {
		s.soft[r] = false
	}
	for c := from; c < to && c < s.cols; c++ {
		s.grid[r][c] = Cell{Rune: ' '}
	}
}

// sgr applies an SGR parameter list to the pen. Only the attributes Cell
// records are tracked; anything else in the list is skipped.
func (s *Screen) sgr(params string) {
	if params == "" {
		s.pen = Cell{}
		return
	}
	parts := strings.Split(params, ";")
	num := func(i int) (int, bool) {
		if i >= len(parts) {
			return 0, false
		}
		n, err := strconv.Atoi(parts[i])
		return n, err == nil
	}
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if strings.HasPrefix(p, "4:") { // 4:n underline style
			s.pen.Underline = p != "4:0"
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			if p == "" {
				s.pen = Cell{}
			}
			continue
		}
		switch {
		case n == 0:
			s.pen = Cell{}
		case n == 1:
			s.pen.Bold = true
		case n == 2:
			s.pen.Dim = true
		case n == 3:
			s.pen.Italic = true
		case n == 4:
			s.pen.Underline = true
		case n == 7:
			s.pen.Reverse = true
		case n == 9:
			s.pen.Strike = true
		case n == 22:
			s.pen.Bold, s.pen.Dim = false, false
		case n == 23:
			s.pen.Italic = false
		case n == 24:
			s.pen.Underline = false
		case n == 27:
			s.pen.Reverse = false
		case n == 29:
			s.pen.Strike = false
		case n >= 30 && n <= 37:
			s.pen.FG = ansi.BasicColor(n - 30)
		case n >= 90 && n <= 97:
			s.pen.FG = ansi.BasicColor(n - 90 + 8)
		case n == 39:
			s.pen.FG = nil
		case n >= 40 && n <= 47:
			s.pen.BG = ansi.BasicColor(n - 40)
		case n >= 100 && n <= 107:
			s.pen.BG = ansi.BasicColor(n - 100 + 8)
		case n == 49:
			s.pen.BG = nil
		case n == 38 || n == 48:
			var c ansi.Color
			if k, ok := num(i + 1); ok && k == 5 {
				if v, ok := num(i + 2); ok {
					c = ansi.Color256(clampByte(v))
				}
				i += 2
			} else if ok && k == 2 {
				r, _ := num(i + 2)
				g, _ := num(i + 3)
				b, _ := num(i + 4)
				c = ansi.RGB{R: clampByte(r), G: clampByte(g), B: clampByte(b)}
				i += 4
			}
			if n == 38 {
				s.pen.FG = c
			} else {
				s.pen.BG = c
			}
		}
	}
}

// clampByte limits n to 0..255 for an SGR colour component.
func clampByte(n int) uint8 {
	return uint8(min(max(n, 0), 255)) // #nosec G115 -- clamped to the uint8 range just above
}
