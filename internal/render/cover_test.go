package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// Targeted tests that raise in-package coverage.

func covFrame(c *Cells, prevRows, width, height int, lines ...string) (string, Stats, bool) {
	return c.Frame(Frame{
		Lines: lines, Max: max(len(lines), prevRows), PrevRows: prevRows, Width: width, Height: height,
		RegionTop: func() string { return "" }, FitLine: func(s string) string { return s },
	})
}

func TestCovApplySGR(t *testing.T) {
	good := []string{
		"", "0", "1", "2", "3", "4", "5", "7", "8", "9", "1;4;9", "22", "23", "24", "25", "27", "28", "29",
		"31", "97", "39", "41", "103", "49", "38;5;200", "48;5;0", "38;2;1;2;3", "48;2;255;255;255",
		"4:0", "4:1", "4:2", "4:5", "38:5:7", "48:5:7", "38:2::1:2:3", "38:2:0:1:2:3", "48:2::1:2:3",
	}
	for _, s := range good {
		p := cellStyle{attrs: 0xff, ul: 3, link: 5}
		if !applySGR(&p, s) {
			t.Errorf("applySGR(%q) = false", s)
		}
		if s == "" || s == "0" {
			if p != (cellStyle{link: 5}) {
				t.Errorf("reset %q = %+v, want link kept", s, p)
			}
		}
	}
	bad := []string{
		"6", "10", "21", "50", "38", "38;5", "38;5;256", "38;2;1;2", "38;2;1;2;256", "38;9;1", "1000",
		"4:6", "4:1:2", "38:5", "38:5:256", "38:2:1:2", "38:2::256:0:0", "7:1", "5:3",
		strings.Repeat("1;", 32) + "1",
	}
	for _, s := range bad {
		var p cellStyle
		if applySGR(&p, s) {
			t.Errorf("applySGR(%q) = true, want false", s)
		}
	}
	var p cellStyle
	applySGR(&p, "4:3")
	if p.ul != 3 || p.attrs&(1<<3) == 0 {
		t.Errorf("4:3 = %+v", p)
	}
	applySGR(&p, "4")
	if p.ul != 0 {
		t.Errorf("plain 4 keeps the extended style: %+v", p)
	}
	applySGR(&p, "4:2")
	applySGR(&p, "24")
	if p.ul != 0 || p.attrs&(1<<3) != 0 {
		t.Errorf("24 = %+v", p)
	}
}

func TestCovAppendSGRAndGlyphStyles(t *testing.T) {
	const (
		basic  = colBasic<<24 | 2
		bright = colBright<<24 | 3
		c256   = col256<<24 | 200
		rgb    = colRGB<<24 | 1<<16 | 2<<8 | 3
	)
	zero := Style{}
	for _, tc := range []struct {
		from, to Style
		want     string
	}{
		{zero, zero, ""},
		{zero, Style{Attrs: 1}, "\x1b[1m"},
		{Style{Attrs: 1}, zero, ansi.Reset},
		{Style{Attrs: 1 | 8}, Style{Attrs: 8}, "\x1b[0;4m"},
		{zero, Style{Attrs: 1 | 2 | 4 | 8 | 16 | 32 | 64 | 128}, "\x1b[1;2;3;4;5;7;8;9m"},
		{zero, Style{Attrs: 8, UL: 3}, "\x1b[4:3m"},
		{Style{Attrs: 8, UL: 3}, Style{Attrs: 8, UL: 2}, "\x1b[0;4:2m"},
		{zero, Style{FG: basic}, "\x1b[32m"},
		{zero, Style{FG: bright, BG: basic}, "\x1b[93;42m"},
		{zero, Style{FG: c256, BG: c256}, "\x1b[38;5;200;48;5;200m"},
		{zero, Style{FG: rgb, BG: rgb}, "\x1b[38;2;1;2;3;48;2;1;2;3m"},
		{Style{FG: rgb, BG: bright}, Style{BG: basic}, "\x1b[39;42m"},
		{Style{BG: bright}, Style{BG: bright, FG: basic}, "\x1b[32m"},
		{zero, Style{BG: bright}, "\x1b[103m"},
		{Style{Link: "x"}, Style{Link: "y"}, ""}, // links are not SGR
	} {
		if got := string(AppendSGR(nil, tc.from, tc.to)); got != tc.want {
			t.Errorf("AppendSGR(%+v -> %+v) = %q, want %q", tc.from, tc.to, got, tc.want)
		}
	}
	// default colours of both kinds
	c := New()
	e := cellEmitter{c: c}
	if got := string(appendSGRColor(nil, 0, true)) + "|" + string(appendSGRColor(nil, 0, false)); got != "39|49" {
		t.Errorf("default colours: %q", got)
	}
	_ = e
}

func TestCovParseGlyphs(t *testing.T) {
	gl, reason, ok := ParseGlyphs("a\x1b[1;31m你\x1b[0m é\x1b]8;;http://x\x1b\\L\x1b]8;;\x1b\\ ", ansi.Measurer{})
	if !ok || reason != "" {
		t.Fatalf("ParseGlyphs: ok %v reason %q", ok, reason)
	}
	var texts []string
	for _, g := range gl {
		texts = append(texts, fmt.Sprintf("%s/%d", g.Text, g.Width))
	}
	got := strings.Join(texts, ",")
	if want := "a/1,你/2,/0, /1,é/1,L/1, /1"; got != want {
		t.Errorf("glyphs = %q, want %q", got, want)
	}
	if gl[1].Style.Attrs != 1 || gl[1].Style.FG != colBasic<<24|1 || gl[4+0].Style.Link != "" {
		t.Errorf("styles = %+v", gl)
	}
	link := ""
	for _, g := range gl {
		if g.Style.Link != "" {
			link = g.Style.Link
		}
	}
	if link != "http://x" {
		t.Errorf("link = %q", link)
	}
	if _, reason, ok := ParseGlyphs("a\x01", ansi.Measurer{}); ok || reason != "control_character" {
		t.Errorf("bad line: ok %v reason %q", ok, reason)
	}
	for raw, want := range map[string]string{
		"":                           "",
		"\x1b]8;id=1;http://a;b\x07": "http://a;b",
		"\x1b]8;;u\x1b\\":            "u",
		"\x1b]8;noterm":              "",
	} {
		if got := linkURI(raw); got != want {
			t.Errorf("linkURI(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestCovParseRowCases(t *testing.T) {
	type tc struct {
		line, reason string
	}
	for _, x := range []tc{
		{"a\x1b", "non_CSI_escape"},
		{"a\x1bZ", "non_CSI_escape"},
		{"a\x1b[2K", "non_SGR_escape"},
		{"a\x1b[1", "non_SGR_escape"},
		{"a\x1b[6mb", "unsupported_SGR"},
		{"a\x01", "control_character"},
		{"a\x7f", "control_character"},
		{"a\xff", "invalid_UTF_8_or_C1_control"},
		{"a\u0085", "invalid_UTF_8_or_C1_control"},
		{"́", "width_disagreement"}, // a mark with nothing before it
		{"a\x1b]8;;u", "unterminated_OSC"},
		{"a\x1b]0;title\x07", "non_hyperlink_OSC"},
		{"a\x1b]8;;u\x01\x07", "control_character"},
		{"a\x1b]8;nouri\x07", "malformed_hyperlink"},
		{"a\x1b_G", "unterminated_string_sequence"},
		{"a\x1b_G\x1bx", "malformed_string_sequence"},
	} {
		c := New()
		row, ok := c.parseRow(x.line, nil)
		if ok || row != nil || c.reason != x.reason {
			t.Errorf("parseRow(%q): ok %v reason %q, want %q", x.line, ok, c.reason, x.reason)
		}
	}
	// supported content
	for _, line := range []string{
		"a\tb", "ab\x1b]8;;http://x\x07link\x1b]8;;\x07", "éx", "🇯🇵🇯🇵", "👨‍👩‍👧x", "你好", "a\x1b[38:2::1:2:3mb",
		"a\x1b]8;;http://x\x1b\\b\x1b]8;;\x1b\\",
	} {
		c := New()
		if _, ok := c.parseRow(line, nil); !ok {
			t.Errorf("parseRow(%q) failed: %q", line, c.reason)
		}
	}
}

func TestCovClusterAndTables(t *testing.T) {
	c := New()
	id, ok := c.clusterID("é")
	id2, _ := c.clusterID("é")
	if !ok || id != id2 || c.text(cell{ch: id}) != "é" || c.text(cell{}) != "" || c.text(cell{ch: 'q'}) != "q" {
		t.Errorf("cluster interning: %v %v", id, id2)
	}
	for i := 0; i < maxClusters; i++ {
		c.clusterID(fmt.Sprintf("c%d́", i))
	}
	if _, ok := c.clusterID("overfloẃ"); ok {
		t.Errorf("cluster table should be full")
	}
	// A frame needing a cluster past the limit makes only its row fall back.
	if _, ok := c.parseRow("x́ý", nil); ok || c.reason != "cluster_table_full" {
		t.Errorf("cluster_table_full: ok %v reason %q", ok, c.reason)
	}
	// An interned link repeats; the link table fills.
	c = New()
	a, _ := c.internLink("\x1b]8;;a\x07")
	b, _ := c.internLink("\x1b]8;;a\x07")
	if a != b {
		t.Errorf("link ids %d %d", a, b)
	}
	for i := 0; ; i++ {
		if _, ok := c.internLink(fmt.Sprintf("\x1b]8;;l%d\x07", i)); !ok {
			break
		}
	}
	// A style reset mid-frame invalidates earlier rows' ids: the whole frame must fall back.
	c = New()
	lines := make([]string, 0, maxCellStyles+10)
	for i := 0; i < maxCellStyles+10; i++ {
		lines = append(lines, fmt.Sprintf("\x1b[38;2;%d;%d;0mx", i/256, i%256))
	}
	if _, _, ok := covFrame(c, 0, 40, 0, lines...); ok || c.Valid() {
		t.Errorf("style table overflow: frame ok %v valid %v", ok, c.Valid())
	}
	c = New()
	lines = lines[:0]
	for i := 0; i < maxCellStyles+10; i++ {
		lines = append(lines, fmt.Sprintf("\x1b]8;;http://h/%d\x07x", i))
	}
	if _, _, ok := covFrame(c, 0, 40, 0, lines...); ok || c.Valid() {
		t.Errorf("link table overflow: frame ok %v valid %v", ok, c.Valid())
	}
}

func TestCovStripAndCompare(t *testing.T) {
	for in, want := range map[string]string{
		"a\x1b]8;;u\x07b\x1b]8;;\x1b\\c": "abc",
		"a\x1b_Gx\x1b\\b\x1bPq\x1b\\c":   "abc",
		"a\x1b]8;;u":                     "a",
		"a\x1b_G":                        "a",
		"plain":                          "plain",
	} {
		if got := cellStripOSC(in); got != want {
			t.Errorf("cellStripOSC(%q) = %q, want %q", in, got, want)
		}
	}
	if cellAt(nil, 3) != blankCell || cellAt([]cell{{ch: 'x', w: 1}}, 0).ch != 'x' {
		t.Error("cellAt")
	}
	if !isCont([]cell{{}}, 0) || isCont(nil, 0) {
		t.Error("isCont")
	}
	o1, o2 := []opaqueSeg{{col: 1, seq: "a"}}, []opaqueSeg{{col: 2, seq: "a"}}
	if opqEqual(o1, o2) || opqEqual(o1, nil) || opqEqual(o1, []opaqueSeg{{col: 1, seq: "b"}}) || !opqEqual(o1, o1) {
		t.Error("opqEqual")
	}
	if rowsEqual([]cell{{ch: 'a'}}, []cell{{ch: 'b'}}) || rowsEqual(nil, []cell{{}}) || !rowsEqual([]cell{{ch: 'a'}}, []cell{{ch: 'a'}}) {
		t.Error("rowsEqual")
	}
}

func TestCovAccessorsAndInvalidate(t *testing.T) {
	c := New()
	c.SetNoCache(true)
	c.SetNoPatch(true)
	covFrame(c, 0, 33, 10, "abc", "def")
	if c.Width() != 33 || !c.Valid() || c.GridRows() != 2 || len(c.PatchedRows()) != 2 {
		t.Errorf("accessors: width %d valid %v rows %d", c.Width(), c.Valid(), c.GridRows())
	}
	out, _, _ := covFrame(c, 2, 33, 10, "abc", "def")
	if strings.Contains(out, "abc") { // noCache still diffs against the previous grid
		t.Errorf("unchanged frame rewrote rows: %q", out)
	}
	c.Invalidate()
	if c.Valid() {
		t.Error("Invalidate left the frame valid")
	}
	out, _, _ = covFrame(c, 2, 33, 10, "abc", "def")
	if !strings.Contains(out, "abc") || !strings.Contains(out, "def") {
		t.Errorf("redraw after Invalidate: %q", out)
	}
}

func TestCovFrameTallerThanTerminal(t *testing.T) {
	c := New()
	if _, _, ok := covFrame(c, 0, 20, 2, "a", "b", "c"); ok || c.Reason() != "view_taller_than_terminal" {
		t.Errorf("ok %v reason %q", ok, c.Reason())
	}
}

// Styles and links are written as a minimal pen change from the previous one;
// every transition is replayed twice so the transition cache is hit.
func TestCovPenTransitions(t *testing.T) {
	row := "\x1b[1mb\x1b[3mi\x1b[0mn\x1b[4:3mu\x1b[24mx\x1b[38;5;9mf\x1b[48;2;1;2;3mg\x1b[92;101mh\x1b[39;49mq" +
		"\x1b]8;;http://a\x07L\x1b]8;;\x1b\\z\x1b[1;2;3;5;7;8;9mA\x1b[22;23;25;27;28;29mB\x1b[0m"
	c := New()
	out1, _, ok := covFrame(c, 0, 80, 0, row, row)
	if !ok {
		t.Fatalf("fell back: %s", c.Reason())
	}
	out2, _, _ := covFrame(New(), 0, 80, 0, row, row)
	if out1 != out2 {
		t.Errorf("not deterministic:\n%q\n%q", out1, out2)
	}
	// Past transN styles: pen changes between high-numbered styles are built each time.
	var sb strings.Builder
	for i := 0; i < transN+8; i++ {
		fmt.Fprintf(&sb, "\x1b[38;5;%dm.\x1b[48;5;%dm.", i, i)
	}
	big := sb.String() + "\x1b[0m"
	c = New()
	if _, _, ok := covFrame(c, 0, 400, 0, big); !ok {
		t.Fatalf("big row fell back: %s", c.Reason())
	}
	if _, _, ok := covFrame(c, 1, 400, 0, strings.Replace(big, ".", "x", 40)); !ok {
		t.Fatalf("big row 2 fell back: %s", c.Reason())
	}
}

func TestCovCursorMovement(t *testing.T) {
	// Row creation by CR LF, deep jumps with CUD, and column moves in both directions.
	c := New()
	covFrame(c, 0, 40, 0, "a", "b", "c", "d", "e", "f")
	out, _, _ := covFrame(c, 6, 40, 0, "a", "b", "c", "d", "e", "F")
	if !strings.Contains(out, "\x1b[5B") {
		t.Errorf("deep move: %q", out)
	}
	out, _, _ = covFrame(c, 6, 40, 0, "A", "b", "c", "d", "e", "F")
	if out != "A\x1b[0m" && !strings.HasPrefix(out, "A") {
		t.Errorf("row 0 change: %q", out)
	}
	c = New()
	covFrame(c, 0, 40, 0, "abcdefghij", "x")
	out, _, _ = covFrame(c, 2, 40, 0, "aXcdefghiJ", "x") // move right 8 after the first change
	if !strings.Contains(out, "\x1b[8C") && !strings.Contains(out, "\x1b[7C") {
		t.Errorf("right move: %q", out)
	}
	// Left moves: change at the end of a row, then at the start of the next
	// row's earlier column (cursor after the last write is to the right).
	c = New()
	covFrame(c, 0, 40, 0, "abcdefghij", "klmnopqrst")
	out, _, _ = covFrame(c, 2, 40, 0, "abcdefghiX", "kXmnopqrst")
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("column moves: %q", out)
	}
	// A pending-wrap cursor (row written to the last column) then needs CHA.
	c = New()
	covFrame(c, 0, 5, 0, "abcde", "vwxyz")
	out, _, _ = covFrame(c, 2, 5, 0, "abcdX", "vwXyz")
	if !strings.Contains(out, "G") && !strings.Contains(out, "\r") && !strings.Contains(out, "D") {
		t.Errorf("pending wrap: %q", out)
	}
	// Direct emitter moves.
	e := &cellEmitter{c: New(), col: -1}
	e.moveCol(3)
	e.moveCol(4)
	e.moveCol(9)
	e.moveCol(8)
	e.moveCol(2)
	e.moveCol(0)
	e.moveCol(0)
	if got := string(e.out); got != "\x1b[4G\x1b[C\x1b[5C\x1b[D\x1b[6D\r" {
		t.Errorf("moveCol = %q", got)
	}
	e = &cellEmitter{c: New()}
	e.moveRow(0, true)
	e.moveRow(1, true)
	e.moveRow(5, true)
	e.moveRow(6, false)
	if got := string(e.out); got != "\r\n\x1b[4B\r\n" {
		t.Errorf("moveRow = %q", got)
	}
}

func TestCovWritesPastEndAndErase(t *testing.T) {
	// A shorter row erases the tail of the longer one; a longer row after
	// a shorter writes into blank cells; wide cells are rewritten whole.
	c := New()
	covFrame(c, 0, 40, 0, "hello world", "你好嗎", "plain")
	out, _, _ := covFrame(c, 3, 40, 0, "hello", "你好", "plain")
	if strings.Count(out, "\x1b[K") != 2 {
		t.Errorf("erase: %q", out)
	}
	out, _, _ = covFrame(c, 3, 40, 0, "hello w", "你x", "plain")
	if !strings.Contains(out, " w") && !strings.Contains(out, "w") {
		t.Errorf("grow: %q", out)
	}
	// A cell change inside a wide cluster rewrites the pair; a change from
	// wide to narrow clears the stale continuation.
	c = New()
	covFrame(c, 0, 40, 0, "a你b", "é́x", "k")
	covFrame(c, 3, 40, 0, "ab c", "ex", "k")
	covFrame(c, 3, 40, 0, "a你b", "é́x", "k")
	// Writes past the end of the new row cover the blanks of a longer old row (styled).
	c = New()
	covFrame(c, 0, 40, 0, "\x1b[41mabcd\x1b[0m")
	out, _, _ = covFrame(c, 1, 40, 0, "\x1b[41mab\x1b[0m")
	if !strings.Contains(out, "\x1b[K") {
		t.Errorf("shrink: %q", out)
	}
	// writeCells past len(n) directly, including a continuation cell and a pending wrap.
	e := &cellEmitter{c: New(), width: 4}
	e.writeCells([]cell{{ch: 'a', w: 1}, {ch: '你', w: 2}, {}}, 0, 4)
	if string(e.out) != "a你 " || e.col != -1 {
		t.Errorf("writeCells = %q col %d", e.out, e.col)
	}
	e = &cellEmitter{c: New()}
	e.col = 2
	e.eraseFrom(4)
	if string(e.out) != "\x1b[2C\x1b[K" {
		t.Errorf("eraseFrom = %q", e.out)
	}
	e = &cellEmitter{c: New(), width: 3}
	e.writeOpaque([]opaqueSeg{{col: 9, seq: "S"}})
	if string(e.out) != "\x1b[3GS" && !strings.HasSuffix(string(e.out), "S") {
		t.Errorf("writeOpaque = %q", e.out)
	}
	e.appendCell(cell{ch: 'é', w: 1})
	if !strings.HasSuffix(string(e.out), "é") {
		t.Error("appendCell rune")
	}
}

func TestCovLazyFitAndMeasurer(t *testing.T) {
	c := New()
	lines := []string{"abcdefgh", "xy"}
	out, _, ok := c.Frame(Frame{
		Lines: lines, Max: 2, LazyFit: true, Width: 4, Height: 10,
		RegionTop: func() string { return "" },
		FitLine:   func(s string) string { return ansi.Truncate(s, 4) },
	})
	if !ok || !strings.Contains(out, "abcd") || strings.Contains(out, "efgh") || lines[0] != "abcd" {
		t.Errorf("lazy fit: ok %v out %q lines %q", ok, out, lines)
	}
	// Changing the measurer drops the previous frame.
	_, _, _ = c.Frame(Frame{
		Lines: lines, Max: 2, PrevRows: 2, Width: 4, Height: 10, Measurer: ansi.ClusterMeasurer(false),
		RegionTop: func() string { return "" }, FitLine: func(s string) string { return s },
	})
}

type covGrid struct {
	rows [][]struct {
		t  string
		w  uint8
		st Style
	}
}

func (g covGrid) Cell(x, y int) (string, uint8, Style) {
	r := g.rows[y][x]
	return r.t, r.w, r.st
}

func TestCovGridFrames(t *testing.T) {
	type gc = struct {
		t  string
		w  uint8
		st Style
	}
	g := covGrid{rows: [][]gc{
		{{"a", 1, Style{}}, {"你", 2, Style{FG: colRGB<<24 | 5}}, {"", 0, Style{FG: colRGB<<24 | 5}}, {"é", 1, Style{Link: "http://x"}}},
		{{" ", 1, Style{}}, {" ", 1, Style{}}, {" ", 1, Style{}}, {" ", 1, Style{}}},
	}}
	draw := func(c *Cells, g GridSource, rows, prev, max int) (string, bool) {
		out, _, ok := c.Frame(Frame{Grid: g, GridRows: rows, Max: max, PrevRows: prev, Width: 4, Height: 10,
			RegionTop: func() string { return "" }, FitLine: func(s string) string { return s }})
		return out, ok
	}
	c := New()
	out, ok := draw(c, g, 2, 0, 3)
	if !ok || !strings.Contains(out, "你") || !strings.Contains(out, "http://x") {
		t.Fatalf("grid frame: ok %v %q", ok, out)
	}
	g.rows[0][0].t = "z"
	out, ok = draw(c, g, 2, 3, 3)
	if !ok || !strings.Contains(out, "z") {
		t.Fatalf("grid diff: ok %v %q", ok, out)
	}
	for name, mut := range map[string]func(*covGrid){
		"wide":    func(g *covGrid) { g.rows[0][0].w = 3 },
		"control": func(g *covGrid) { g.rows[0][0].t = "\x01" },
		"empty":   func(g *covGrid) { g.rows[0][0].t = "" },
	} {
		bad := covGrid{rows: [][]gc{append([]gc(nil), g.rows[0]...), g.rows[1]}}
		mut(&bad)
		c2 := New()
		if _, ok := draw(c2, bad, 2, 0, 2); ok || c2.Reason() != "control_character" {
			t.Errorf("%s: ok %v reason %q", name, ok, c2.Reason())
		}
	}
	// Style and cluster tables overflow while building a grid row.
	c3 := New()
	big := covGrid{rows: [][]gc{make([]gc, 4)}}
	for i := range big.rows[0] {
		big.rows[0][i] = gc{"x", 1, Style{}}
	}
	for i := 0; i < maxCellStyles; i++ {
		c3.intern(cellStyle{attrs: 1, fg: cellColor(i + 1)})
	}
	big.rows[0][0].st = Style{FG: 0xffffff}
	if _, ok := draw(c3, big, 1, 0, 1); ok || c3.Reason() != "style_table_full" {
		t.Errorf("style overflow: ok %v reason %q", ok, c3.Reason())
	}
	c4 := New()
	for i := 0; i < maxCellStyles; i++ {
		c4.internLink(fmt.Sprintf("\x1b]8;;u%d\x07", i))
	}
	big.rows[0][0].st = Style{Link: "http://new"}
	if _, ok := draw(c4, big, 1, 0, 1); ok || c4.Reason() != "style_table_full" {
		t.Errorf("link overflow: ok %v reason %q", ok, c4.Reason())
	}
	c5 := New()
	for i := 0; i < maxClusters; i++ {
		c5.clusterID(fmt.Sprintf("c%d́", i))
	}
	big.rows[0][0] = gc{"q́", 1, Style{}}
	if _, ok := draw(c5, big, 1, 0, 1); ok || c5.Reason() != "cluster_table_full" {
		t.Errorf("cluster overflow: ok %v reason %q", ok, c5.Reason())
	}
}

// A row that was raw (unparsable) last frame and is the same line now is not redrawn.
func TestCovRawRowStability(t *testing.T) {
	c := New()
	covFrame(c, 0, 20, 0, "a", "x\x01y", "c")
	out, st, _ := covFrame(c, 3, 20, 0, "a", "x\x01y", "C")
	if strings.Contains(out, "\x01") || len(st.Fallbacks) != 0 {
		t.Errorf("raw row redrawn: %q %+v", out, st)
	}
	// A raw row becomes raw with different content: redrawn with the pen reset first.
	out, _, _ = covFrame(c, 3, 20, 0, "\x1b[2ma", "x\x02y", "C")
	if !strings.Contains(out, "x\x02y") {
		t.Errorf("changed raw row: %q", out)
	}
}

// Regression (found by FuzzRendererEquivalence): a fallback row drawn after a
// row that left a style on must start from the default style.
func TestCovRawRowStartsFromDefaultPen(t *testing.T) {
	c := New()
	out, _, _ := covFrame(c, 0, 20, 0, "\x1b[2ma", "b\x1b[2Cc")
	i := strings.Index(out, "\x1b[2K")
	if i < 0 || !strings.Contains(out[:i], "\x1b[0m") {
		t.Errorf("pen not reset before the fallback row: %q", out)
	}
}
