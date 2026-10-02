package tui

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// Equivalence tests for the opt-in cell renderer (spec #40, decision #4).
// vtScreen is a minimal terminal model that keeps per-cell style (unlike
// visibleLines in render_test.go), so a lost or leaked style is caught.

type vtStyle struct {
	attrs  uint16
	fg, bg string
}

type vtCell struct {
	s  string
	w  int // 1, 2, or 0 for the trailing half of a wide cell
	st vtStyle
}

type vtScreen struct {
	t        *testing.T
	width    int
	rows     [][]vtCell
	row, col int
	pen      vtStyle
}

func newVTScreen(t *testing.T, width int) *vtScreen {
	v := &vtScreen{t: t, width: width}
	v.rows = [][]vtCell{v.blankRow()}
	return v
}

func (v *vtScreen) blankRow() []vtCell {
	r := make([]vtCell, v.width)
	for i := range r {
		r[i] = vtCell{s: " ", w: 1}
	}
	return r
}

func (v *vtScreen) ensureRow(r int) {
	for len(v.rows) <= r {
		v.rows = append(v.rows, v.blankRow())
	}
}

// eraseCell blanks a cell the way an erase does, with the pen's background.
func (v *vtScreen) eraseCell(c int) {
	v.clearWide(c)
	v.rows[v.row][c] = vtCell{s: " ", w: 1, st: vtStyle{bg: v.pen.bg}}
}

// clearWide removes the other half of a wide cell that column c belongs to.
func (v *vtScreen) clearWide(c int) {
	r := v.rows[v.row]
	if r[c].w == 0 && c > 0 {
		r[c-1] = vtCell{s: " ", w: 1}
	}
	if r[c].w == 2 && c+1 < v.width {
		r[c+1] = vtCell{s: " ", w: 1}
	}
}

func (v *vtScreen) feed(data []byte) {
	i := 0
	for i < len(data) {
		b := data[i]
		switch {
		case b == 0x1b:
			if i+1 >= len(data) || data[i+1] != '[' {
				v.t.Fatalf("unsupported escape at %q", data[i:])
			}
			j := i + 2
			for j < len(data) && data[j] >= 0x30 && data[j] <= 0x3f {
				j++
			}
			if j >= len(data) {
				v.t.Fatalf("truncated CSI %q", data[i:])
			}
			v.csi(string(data[i+2:j]), data[j])
			i = j + 1
		case b == '\r':
			v.col = 0
			i++
		case b == '\n':
			v.row++
			v.ensureRow(v.row)
			i++
		case b < 0x20 || b == 0x7f:
			v.t.Fatalf("unsupported control byte %#x", b)
		default:
			r, size := utf8.DecodeRune(data[i:])
			v.print(string(r), ansi.Width(string(r)))
			i += size
		}
	}
}

func (v *vtScreen) print(s string, w int) {
	v.ensureRow(v.row)
	row := v.rows[v.row]
	if w == 0 {
		// Combine with the cell written last.
		c := v.col - 1
		for c > 0 && row[c].w == 0 {
			c--
		}
		if v.col == 0 || c < 0 {
			v.t.Fatalf("combining rune with nothing before it")
		}
		if row[c].s == " " && row[c].st == (vtStyle{}) {
			v.t.Fatalf("combining rune after a blank")
		}
		row[c].s += s
		return
	}
	if v.col+w > v.width {
		v.t.Fatalf("write of %q at col %d overflows width %d", s, v.col, v.width)
	}
	for k := 0; k < w; k++ {
		v.clearWide(v.col + k)
	}
	row[v.col] = vtCell{s: s, w: w, st: v.pen}
	if w == 2 {
		row[v.col+1] = vtCell{w: 0, st: v.pen}
	}
	v.col += w
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return n
}

func (v *vtScreen) csi(params string, final byte) {
	if strings.HasPrefix(params, "?") {
		if params == "?2026" && (final == 'h' || final == 'l') {
			return
		}
		v.t.Fatalf("unsupported private CSI %q%c", params, final)
	}
	n := atoiDefault(params, 1)
	if n == 0 {
		n = 1
	}
	pending := v.col >= v.width // wrapped-pending state after the last column
	switch final {
	case 'A':
		v.row -= n
		if v.row < 0 {
			v.row = 0
		}
	case 'B':
		if v.row+n >= len(v.rows) {
			v.t.Fatalf("CUD %d from row %d leaves the %d known rows", n, v.row, len(v.rows))
		}
		v.row += n
	case 'C':
		if pending {
			v.col = v.width - 1
		}
		v.col += n
		if v.col > v.width-1 {
			v.col = v.width - 1
		}
	case 'D':
		if pending {
			v.col = v.width - 1
		}
		v.col -= n
		if v.col < 0 {
			v.col = 0
		}
	case 'G':
		v.col = n - 1
		if v.col > v.width-1 {
			v.col = v.width - 1
		}
	case 'K':
		v.ensureRow(v.row)
		c := v.col
		if pending {
			c = v.width - 1
		}
		switch atoiDefault(params, 0) {
		case 0:
			for k := c; k < v.width; k++ {
				v.eraseCell(k)
			}
		case 1:
			for k := 0; k <= c; k++ {
				v.eraseCell(k)
			}
		case 2:
			for k := 0; k < v.width; k++ {
				v.eraseCell(k)
			}
		}
	case 'J':
		if atoiDefault(params, 0) != 0 {
			v.t.Fatalf("unsupported ED %q", params)
		}
		v.ensureRow(v.row)
		c := v.col
		if pending {
			c = v.width - 1
		}
		for k := c; k < v.width; k++ {
			v.eraseCell(k)
		}
		v.rows = v.rows[:v.row+1]
	case 'm':
		v.sgr(params)
	default:
		v.t.Fatalf("unsupported CSI %q%c", params, final)
	}
}

func (v *vtScreen) sgr(params string) {
	if params == "" {
		v.pen = vtStyle{}
		return
	}
	parts := strings.Split(params, ";")
	num := func(k int) int {
		if k >= len(parts) {
			v.t.Fatalf("short SGR %q", params)
		}
		return atoiDefault(parts[k], 0)
	}
	for k := 0; k < len(parts); k++ {
		c := num(k)
		bit := map[int]uint16{1: 1, 2: 2, 3: 4, 4: 8, 5: 16, 7: 32, 8: 64, 9: 128}
		clr := map[int]uint16{22: 3, 23: 4, 24: 8, 25: 16, 27: 32, 28: 64, 29: 128}
		switch {
		case c == 0:
			v.pen = vtStyle{}
		case bit[c] != 0:
			v.pen.attrs |= bit[c]
		case clr[c] != 0:
			v.pen.attrs &^= clr[c]
		case c >= 30 && c <= 37 || c >= 90 && c <= 97:
			v.pen.fg = "n" + strconv.Itoa(c)
		case c == 39:
			v.pen.fg = ""
		case c >= 40 && c <= 47 || c >= 100 && c <= 107:
			v.pen.bg = "n" + strconv.Itoa(c-10)
		case c == 49:
			v.pen.bg = ""
		case c == 38 || c == 48:
			var s string
			if num(k+1) == 5 {
				s = "5:" + strconv.Itoa(num(k+2))
				k += 2
			} else {
				s = fmt.Sprintf("2:%d,%d,%d", num(k+2), num(k+3), num(k+4))
				k += 4
			}
			if c == 38 {
				v.pen.fg = s
			} else {
				v.pen.bg = s
			}
		default:
			v.t.Fatalf("unsupported SGR %d in %q", c, params)
		}
	}
}

// equalScreens reports the first difference between two screens (blank rows
// pad the shorter one), comparing cells, styles and the cursor row.
func equalScreens(a, b *vtScreen) string {
	n := len(a.rows)
	if len(b.rows) > n {
		n = len(b.rows)
	}
	a.ensureRow(n - 1)
	b.ensureRow(n - 1)
	for r := 0; r < n; r++ {
		for c := 0; c < a.width; c++ {
			if a.rows[r][c] != b.rows[r][c] {
				return fmt.Sprintf("cell (%d,%d): %+v vs %+v", r, c, a.rows[r][c], b.rows[r][c])
			}
		}
	}
	if a.row != b.row {
		return fmt.Sprintf("cursor row %d vs %d", a.row, b.row)
	}
	return ""
}

// --- random views ---

var (
	equivStyles = []string{
		"\x1b[1m", "\x1b[3m", "\x1b[31m", "\x1b[38;5;123m", "\x1b[38;2;10;200;30m",
		"\x1b[44m", "\x1b[48;5;9m", "\x1b[92m", "\x1b[22m", "\x1b[39m", "\x1b[49m",
		"\x1b[0m", "\x1b[1;4;33m", "\x1b[7m", "\x1b[m", "\x1b[48;2;1;2;3m",
	}
	equivTokens = []string{
		"a", "b", "c", "x", "Z", "1", " ", " ", " ", "é", "你", "世", "好", "😀", "ö",
		"你́", "~",
	}
)

func randomLine(r *rand.Rand, width int) string {
	var sb strings.Builder
	used := 0
	styled := false
	for used < width && r.Intn(12) != 0 {
		if r.Intn(4) == 0 {
			sb.WriteString(equivStyles[r.Intn(len(equivStyles))])
			styled = true
			continue
		}
		pool := equivTokens
		if width%2 == 0 {
			pool = equivTokens[:9] // even widths: ASCII only, which the cell renderer can patch
		}
		tok := pool[r.Intn(len(pool))]
		w := ansi.Width(tok)
		if used+w > width {
			break
		}
		sb.WriteString(tok)
		used += w
	}
	if styled {
		sb.WriteString("\x1b[0m")
	}
	return sb.String()
}

func randomView(r *rand.Rand, width, rows int) []string {
	lines := make([]string, rows)
	for i := range lines {
		lines[i] = randomLine(r, width)
	}
	return lines
}

// nextView derives the next frame from the previous one: identical, one row
// redrawn, one cell swapped, more or fewer rows, or a new view.
func nextView(r *rand.Rand, prev []string, width int) []string {
	next := append([]string(nil), prev...)
	switch r.Intn(9) {
	case 0: // identical
	case 1: // one row redrawn
		next[r.Intn(len(next))] = randomLine(r, width)
	case 2: // grow
		for k := r.Intn(3) + 1; k > 0 && len(next) < 12; k-- {
			next = append(next, randomLine(r, width))
		}
	case 3: // shrink
		if len(next) > 1 {
			next = next[:1+r.Intn(len(next)-1)]
		}
	case 4: // brand new
		next = randomView(r, width, 1+r.Intn(12))
	case 7, 8: // swap a few printable ASCII characters of a row in place (the cell renderer's patch path)
		i := r.Intn(len(next))
		b := []byte(next[i])
		for m := 1 + r.Intn(3); m > 0 && len(b) > 0; m-- {
			k := r.Intn(len(b))
			if b[k] > 0x20 && b[k] < 0x7f && !inEscape(b, k) {
				b[k] = byte('a' + r.Intn(26))
			}
		}
		next[i] = string(b)
	default: // edit part of a row: keep a prefix or suffix, redraw the rest
		i := r.Intn(len(next))
		other := randomLine(r, width)
		if r.Intn(2) == 0 {
			next[i] = ansi.Truncate(next[i], r.Intn(width+1)) + "\x1b[0m" + ansi.TrimLeftWidth(other, r.Intn(width+1))
			if ansi.Width(next[i]) > width {
				next[i] = ansi.Truncate(next[i], width)
			}
		} else {
			next[i] = other
		}
	}
	return next
}

func equivProgram(buf *bytes.Buffer, width int, cell bool) *Program {
	p := NewProgram(staticModel{}, WithOutput(buf), WithColorProfile(ansi.TrueColor), WithCellRenderer(cell))
	p.width, p.height = width, 100
	return p
}

// referenceScreen feeds the frames' rows straight to a fresh screen.
func referenceScreen(t *testing.T, width, region int, lines []string) *vtScreen {
	v := newVTScreen(t, width)
	for i := 0; i < region; i++ {
		if i > 0 {
			v.feed([]byte("\r\n"))
		}
		if i < len(lines) {
			v.feed([]byte(lines[i]))
		}
		v.feed([]byte("\x1b[0m"))
	}
	return v
}

func TestCellRendererEquivalentToLineRenderer(t *testing.T) {
	const sequences, frames = 200, 50 // 10,000 random views
	fallbacks, patchedRows := 0, 0
	for seq := 0; seq < sequences; seq++ {
		r := rand.New(rand.NewSource(int64(seq) + 1))
		width := 8 + r.Intn(33)
		var lineOut, cellOut bytes.Buffer
		lp, cp := equivProgram(&lineOut, width, false), equivProgram(&cellOut, width, true)
		lv, cv := newVTScreen(t, width), newVTScreen(t, width)
		view := randomView(r, width, 1+r.Intn(12))
		region := 0
		for f := 0; f < frames; f++ {
			if f > 0 {
				view = nextView(r, view, width)
			}
			if len(view) > region {
				region = len(view)
			}
			lp.model = staticModel{view: strings.Join(view, "\n")}
			cp.model = lp.model
			lp.render()
			cp.render()
			lv.feed(lineOut.Bytes())
			cv.feed(cellOut.Bytes())
			lineOut.Reset()
			cellOut.Reset()

			ref := referenceScreen(t, width, region, view)
			if d := equalScreens(cv, lv); d != "" {
				t.Fatalf("seq %d frame %d: cell vs line renderer: %s\nview %q", seq, f, d, view)
			}
			if d := equalScreens(cv, ref); d != "" {
				t.Fatalf("seq %d frame %d: cell renderer vs reference: %s\nview %q", seq, f, d, view)
			}
			if cp.liveLines != lp.liveLines || len(cp.lastFrame) != len(lp.lastFrame) {
				t.Fatalf("seq %d frame %d: bookkeeping differs: %d/%d vs %d/%d",
					seq, f, cp.liveLines, len(cp.lastFrame), lp.liveLines, len(lp.lastFrame))
			}
			if !cp.cells.Valid() {
				fallbacks++
			}
			for _, ok := range cp.cells.PatchedRows() {
				if ok {
					patchedRows++
				}
			}
		}
	}
	if patchedRows < 300 {
		t.Fatalf("only %d rows took the patch path; the random views do not exercise it", patchedRows)
	}
	if fallbacks != 0 {
		t.Fatalf("%d frames fell back to the line renderer; the generator only uses supported content", fallbacks)
	}
}

// A row the grid cannot represent is drawn alone with the line strategy: the
// frame stays a cell frame and the screen matches the line renderer's.
func TestCellRendererFallsBackForUnsupportedRowOnly(t *testing.T) {
	for _, bad := range []string{"a\x1b[2Cb"} {
		view := "one\n" + bad + "\nthree"
		var buf, want bytes.Buffer
		p := equivProgram(&buf, 40, true)
		q := equivProgram(&want, 40, false)
		cv, lv := newVTScreen(t, 40), newVTScreen(t, 40)
		for _, v := range []string{view, "ONE\n" + bad + "\nthree", "ONE\nTWO\nthree"} {
			p.model = staticModel{view: v}
			q.model = p.model
			p.render()
			q.render()
			if !p.cells.Valid() {
				t.Errorf("%q: the whole frame fell back (%q)", v, p.cells.Reason())
			}
			cv.feed(buf.Bytes())
			lv.feed(want.Bytes())
			buf.Reset()
			want.Reset()
			if d := equalScreens(cv, lv); d != "" {
				t.Errorf("%q: cell vs line renderer: %s", v, d)
			}
		}
	}
}

func TestCellRendererRecoversAfterInvalidation(t *testing.T) {
	var buf bytes.Buffer
	p := equivProgram(&buf, 20, true)
	step := func(view string, wantGrid bool) {
		t.Helper()
		p.model = staticModel{view: view}
		p.render()
		if p.cells.Valid() != wantGrid {
			t.Fatalf("%q: havePrev = %v, want %v", view, p.cells.Valid(), wantGrid)
		}
	}
	step("one\ntwo\nthree", true)
	step("one\nTWO\nthree", true)
	step("bad\x01tab\nTWO\nthree", true) // row 0 drawn with the line strategy, the rest from the grid
	step("bad tab\nTWO\nthree", true)
	p.lastFrame, p.liveLines = nil, 0 // Println, Suspend or a resize
	step("x", true)
	if p.cells.GridRows() != 1 {
		t.Fatalf("grid has %d rows after invalidation, want 1", p.cells.GridRows())
	}
}

func TestCellRendererWritesFewerBytesForOneChangedCell(t *testing.T) {
	const width = 60
	base := make([]string, 8)
	for i := range base {
		base[i] = "\x1b[32m" + strings.Repeat("row content ", 5) + "\x1b[0m"
	}
	changed := append([]string(nil), base...)
	changed[4] = "\x1b[32m" + strings.Repeat("row content ", 2) + "row contenX " + strings.Repeat("row content ", 2) + "\x1b[0m"

	sizes := map[bool]int{}
	screens := map[bool]*vtScreen{}
	for _, cell := range []bool{false, true} {
		var buf bytes.Buffer
		p := equivProgram(&buf, width, cell)
		v := newVTScreen(t, width)
		p.model = staticModel{view: strings.Join(base, "\n")}
		p.render()
		v.feed(buf.Bytes())
		buf.Reset()
		p.model = staticModel{view: strings.Join(changed, "\n")}
		p.render()
		v.feed(buf.Bytes())
		sizes[cell] = buf.Len()
		screens[cell] = v
	}
	if d := equalScreens(screens[true], screens[false]); d != "" {
		t.Fatalf("screens differ: %s", d)
	}
	if !(sizes[true] < sizes[false]) {
		t.Fatalf("cell renderer wrote %d bytes, line renderer %d: want fewer", sizes[true], sizes[false])
	}
	t.Logf("one changed cell: cell renderer %d bytes, line renderer %d bytes", sizes[true], sizes[false])
}

// TestCellRendererOffOutputUnchanged pins the line renderer's bytes: what
// WithLineRenderer and WithCellRenderer(false) write is what releases before
// the cell renderer became the default wrote.
func TestCellRendererOffOutputUnchanged(t *testing.T) {
	// Golden bytes captured from the line renderer.
	on, off := ansi.SyncOutputEnable, ansi.SyncOutputDisable
	views := []string{"a\nbb\nc", "a\nBB\nc", "a"}
	want := []string{
		on + "\r\x1b[2Ka\r\n\x1b[2Kbb\r\n\x1b[2Kc" + off,
		on + "\x1b[2A\r\r\n\x1b[2KBB\r\n" + off,
		on + "\x1b[2A\r\r\n\x1b[2K\r\n\x1b[2K" + off,
	}
	for _, opts := range [][]ProgramOption{{WithCellRenderer(false)}, {WithCellRenderer(false)}} {
		var buf bytes.Buffer
		p := NewProgram(staticModel{}, append([]ProgramOption{WithOutput(&buf), WithColorProfile(ansi.TrueColor)}, opts...)...)
		p.width, p.height = 40, 100
		for i, v := range views {
			p.model = staticModel{view: v}
			p.render()
			if got := buf.String(); got != want[i] {
				t.Fatalf("frame %d: got %q, want %q", i, got, want[i])
			}
			buf.Reset()
		}
		if p.cells != nil {
			t.Fatal("cell renderer state allocated with the option off")
		}
	}
}

// TestCellRendererIsTheDefault proves criterion #72: a Program built without
// renderer options draws with the cell renderer, and WithLineRenderer (or
// WithCellRenderer(false)) restores the line renderer.
func TestCellRendererIsTheDefault(t *testing.T) {
	draw := func(opts ...ProgramOption) *Program {
		var buf bytes.Buffer
		p := NewProgram(staticModel{view: "a\nbb"}, append([]ProgramOption{WithOutput(&buf), WithColorProfile(ansi.TrueColor)}, opts...)...)
		p.width, p.height = 40, 100
		p.render()
		return p
	}
	if p := draw(); !p.cellRender || p.cells == nil {
		t.Fatal("default Program did not use the cell renderer")
	}
	for name, opt := range map[string]ProgramOption{"WithLineRenderer": WithCellRenderer(false), "WithCellRenderer(false)": WithCellRenderer(false)} {
		if p := draw(opt); p.cellRender || p.cells != nil {
			t.Fatalf("%s did not restore the line renderer", name)
		}
	}
}
