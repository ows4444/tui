package vt

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// screenText is the visible rows as text, trailing blanks trimmed.
func screenText(t *Terminal) []string {
	cols, rows := t.Size()
	out := make([]string, rows)
	for y := 0; y < rows; y++ {
		var b strings.Builder
		for x := 0; x < cols; x++ {
			b.WriteString(t.Cell(x, y).Cluster)
		}
		out[y] = strings.TrimRight(b.String(), " ")
	}
	return out
}

func run(cols, rows int, out string) *Terminal {
	t := New(cols, rows)
	t.Write([]byte(out))
	return t
}

func want(t *testing.T, tm *Terminal, rows ...string) {
	t.Helper()
	got := screenText(tm)
	for i := range got {
		w := ""
		if i < len(rows) {
			w = rows[i]
		}
		if got[i] != w {
			t.Errorf("row %d = %q; want %q\nscreen: %q", i, got[i], w, got)
			return
		}
	}
}

func TestPrintWrapAndScroll(t *testing.T) {
	tm := run(5, 3, "abcdefgh")
	want(t, tm, "abcde", "fgh")
	tm = run(5, 3, "one\r\ntwo\r\nthree\r\nfour")
	want(t, tm, "two", "three", "four")
	if tm.Scrollback() != 1 {
		t.Fatalf("scrollback %d", tm.Scrollback())
	}
	if row := tm.Row(1, 0); row[0].Cluster != "o" {
		t.Fatalf("the scrolled-back view starts with %q", row[0].Cluster)
	}
	if row := tm.Row(0, 0); row[0].Cluster != "t" {
		t.Fatalf("the live view starts with %q", row[0].Cluster)
	}
}

func TestPendingWrapIsKeptUntilTheNextGlyph(t *testing.T) {
	tm := run(4, 2, "abcd")
	if x, y, _ := tm.Cursor(); x != 3 || y != 0 {
		t.Fatalf("cursor %d,%d; want it still on the last column", x, y)
	}
	tm.Write([]byte("\rX"))
	want(t, tm, "Xbcd")
	tm = run(4, 2, "abcd\x1b[Dx")
	want(t, tm, "abxd")
}

func TestCursorMovementAndAbsolutePositions(t *testing.T) {
	tm := run(10, 5, "\x1b[3;4HX\x1b[2AY\x1b[3BZ\x1b[2D<\x1b[5C>")
	want(t, tm, "    Y", "", "   X", "    <Z   >", "")
	tm = run(10, 5, "\x1b[3dA\x1b[5GB\x1b[2;2fC")
	want(t, tm, "", " C", "A   B", "", "")
}

func TestEraseDisplayAndLine(t *testing.T) {
	tm := run(5, 3, "aaaaa\r\nbbbbb\r\nccccc\x1b[2;3H\x1b[J")
	want(t, tm, "aaaaa", "bb")
	tm = run(5, 3, "aaaaa\r\nbbbbb\r\nccccc\x1b[2;3H\x1b[1J")
	want(t, tm, "", "   bb", "ccccc")
	tm = run(5, 3, "aaaaa\r\nbbbbb\x1b[2;3H\x1b[K")
	want(t, tm, "aaaaa", "bb")
	tm = run(5, 3, "aaaaa\x1b[1;3H\x1b[1K")
	want(t, tm, "   aa")
	tm = run(5, 3, "aaaaa\x1b[2K")
	want(t, tm)
	tm = run(5, 3, "aaaaa\x1b[1;2H\x1b[2X")
	want(t, tm, "a  aa")
}

func TestInsertAndDeleteLinesAndChars(t *testing.T) {
	tm := run(4, 4, "a\r\nb\r\nc\r\nd\x1b[2;1H\x1b[L")
	want(t, tm, "a", "", "b", "c")
	tm = run(4, 4, "a\r\nb\r\nc\r\nd\x1b[2;1H\x1b[M")
	want(t, tm, "a", "c", "d", "")
	tm = run(6, 1, "abcdef\x1b[1;3H\x1b[2@")
	want(t, tm, "ab  cd")
	tm = run(6, 1, "abcdef\x1b[1;3H\x1b[2P")
	want(t, tm, "abef")
}

func TestScrollRegion(t *testing.T) {
	// Lines 2-3 scroll; the first and last rows stay.
	tm := run(4, 4, "H\r\nA\r\nB\r\nF\x1b[2;3r\x1b[3;1H\n")
	want(t, tm, "H", "B", "", "F")
	if tm.Scrollback() != 0 {
		t.Fatalf("a partial region saved %d lines to scrollback", tm.Scrollback())
	}
	tm = run(4, 4, "H\r\nA\r\nB\r\nF\x1b[2;3r\x1b[2;1H\x1bM")
	want(t, tm, "H", "", "A", "F")
	tm = run(4, 4, "1\r\n2\r\n3\r\n4\x1b[S")
	want(t, tm, "2", "3", "4", "")
	tm = run(4, 4, "1\r\n2\r\n3\r\n4\x1b[T")
	want(t, tm, "", "1", "2", "3")
	// A bad region is ignored and resets to the screen.
	tm = run(4, 4, "\x1b[3;2r")
	if tm.top != 0 || tm.bot != 3 {
		t.Fatalf("region %d-%d", tm.top, tm.bot)
	}
}

func TestSGRAttributesAndColours(t *testing.T) {
	tm := run(20, 1, "\x1b[1;3;4;7;9mA\x1b[22;23;24;27;29mB\x1b[31;42mC\x1b[91;104mD\x1b[38;5;200mE\x1b[48;2;1;2;3mF\x1b[38;2;9;8;7mG\x1b[39;49mH\x1b[0mI")
	a, b := tm.Cell(0, 0), tm.Cell(1, 0)
	if a.Attrs != Bold|Italic|Underline|Reverse|Strike || b.Attrs != 0 {
		t.Errorf("attrs %b then %b", a.Attrs, b.Attrs)
	}
	for _, c := range []struct {
		x      int
		fg, bg ansi.Color
	}{
		{2, ansi.BasicColor(1), ansi.BasicColor(2)},
		{3, ansi.BasicColor(9), ansi.BasicColor(12)},
		{4, ansi.Color256(200), ansi.BasicColor(12)},
		{5, ansi.Color256(200), ansi.RGB{R: 1, G: 2, B: 3}},
		{6, ansi.RGB{R: 9, G: 8, B: 7}, ansi.RGB{R: 1, G: 2, B: 3}},
		{7, nil, nil},
		{8, nil, nil},
	} {
		if got := tm.Cell(c.x, 0); got.FG != c.fg || got.BG != c.bg {
			t.Errorf("cell %d: fg %v bg %v; want %v %v", c.x, got.FG, got.BG, c.fg, c.bg)
		}
	}
	tm = run(10, 1, "\x1b[38:2::10:20:30mA\x1b[48:5:17mB\x1b[58;5;3mC")
	if c := tm.Cell(0, 0); c.FG != (ansi.RGB{R: 10, G: 20, B: 30}) {
		t.Errorf("colon rgb: %v", c.FG)
	}
	if c := tm.Cell(1, 0); c.BG != ansi.Color256(17) {
		t.Errorf("colon 256: %v", c.BG)
	}
	if c := tm.Cell(2, 0); c.Attrs&Blink != 0 {
		t.Error("the 5 of an underline colour was read as blink")
	}
}

func TestErasedCellsKeepTheBackground(t *testing.T) {
	tm := run(4, 1, "\x1b[44m\x1b[K")
	if c := tm.Cell(2, 0); c.BG != ansi.BasicColor(4) || c.Cluster != " " {
		t.Fatalf("erased cell %+v", c)
	}
}

func TestTabsAndBackspace(t *testing.T) {
	tm := run(20, 1, "a\tb\t\tc\x08\x08X")
	want(t, tm, "a       b        X c")
	tm = run(20, 1, "a\tb\x1b[Zc")
	want(t, tm, "a       c")
	tm = run(20, 1, "\x1b[3g\tx")
	want(t, tm, "                   x")
}

func TestWideAndCombiningCharacters(t *testing.T) {
	tm := run(6, 2, "日本éx")
	h, c := tm.Cell(0, 0), tm.Cell(1, 0)
	if h.Cluster != "日" || h.Width != 2 || c.Width != 0 || c.Cluster != "" {
		t.Fatalf("wide cell %+v then %+v", h, c)
	}
	if e := tm.Cell(4, 0); e.Cluster != "é" || tm.Cell(5, 0).Cluster != "x" {
		t.Fatalf("combining mark: %q then %q", e.Cluster, tm.Cell(5, 0).Cluster)
	}
	// A wide character that does not fit wraps whole.
	tm = run(5, 2, "abcd日")
	want(t, tm, "abcd", "日")
	// Overwriting half of a wide character blanks the other half.
	tm = run(4, 1, "日\rx")
	if tm.Cell(0, 0).Cluster != "x" || tm.Cell(1, 0).Cluster != " " {
		t.Fatalf("after overwriting the head: %q %q", tm.Cell(0, 0).Cluster, tm.Cell(1, 0).Cluster)
	}
	tm = run(4, 1, "日\x1b[1;2Hx")
	if tm.Cell(0, 0).Cluster != " " || tm.Cell(1, 0).Cluster != "x" {
		t.Fatalf("after overwriting the tail: %q %q", tm.Cell(0, 0).Cluster, tm.Cell(1, 0).Cluster)
	}
}

func TestUTF8SplitAcrossWrites(t *testing.T) {
	tm := New(4, 1)
	for _, b := range []byte("é日") {
		tm.Write([]byte{b})
	}
	want(t, tm, "é日")
	tm.Write([]byte{0xff, 'x'})
	if got := tm.Cell(3, 0).Cluster; got == "" {
		t.Fatal("an invalid byte lost the cell")
	}
}

func TestSequencesSplitAcrossWrites(t *testing.T) {
	tm := New(10, 2)
	for _, b := range []byte("\x1b[2;3Hhi\x1b]0;a title\x07") {
		tm.Write([]byte{b})
	}
	want(t, tm, "", "  hi")
	if tm.Title() != "a title" {
		t.Fatalf("title %q", tm.Title())
	}
}

func TestLineDrawingCharset(t *testing.T) {
	tm := run(5, 1, "\x1b(0lqk\x1b(Bx")
	want(t, tm, "┌─┐x")
	tm = run(5, 1, "\x1b)0\x0elqk\x0fx")
	want(t, tm, "┌─┐x")
}

func TestAlternateScreen(t *testing.T) {
	tm := run(6, 2, "main\x1b[?1049h\x1b[HALT")
	want(t, tm, "ALT")
	if !tm.AltScreen() {
		t.Fatal("not on the alternate screen")
	}
	tm.Write([]byte("\x1b[?1049l"))
	want(t, tm, "main")
	if x, y, _ := tm.Cursor(); x != 4 || y != 0 {
		t.Fatalf("cursor %d,%d after leaving; want it where it was", x, y)
	}
	// Scrolling on the alternate screen leaves no history.
	tm = run(4, 2, "\x1b[?1049h\x1b[H1\r\n2\r\n3")
	if tm.Scrollback() != 0 {
		t.Fatalf("%d lines of scrollback from the alternate screen", tm.Scrollback())
	}
}

func TestSaveAndRestoreCursor(t *testing.T) {
	tm := run(10, 3, "\x1b[2;4H\x1b7\x1b[1;1HA\x1b8B")
	want(t, tm, "A", "   B")
	tm = run(10, 3, "\x1b[2;4H\x1b[s\x1b[1;1H\x1b[uC")
	want(t, tm, "", "   C")
}

func TestModesAreReported(t *testing.T) {
	tm := run(10, 3, "\x1b[?1h\x1b[?2004h\x1b[?25l\x1b[?1002h\x1b=")
	if !tm.AppCursorKeys() || !tm.BracketedPaste() || !tm.AppKeypad() || tm.Mouse() != MouseDrag {
		t.Fatalf("modes: %v %v %v %v", tm.AppCursorKeys(), tm.BracketedPaste(), tm.AppKeypad(), tm.Mouse())
	}
	if _, _, vis := tm.Cursor(); vis {
		t.Fatal("the cursor is still shown")
	}
	tm.Write([]byte("\x1b[?1l\x1b[?2004l\x1b[?25h\x1b[?1002l\x1b>"))
	if tm.AppCursorKeys() || tm.BracketedPaste() || tm.AppKeypad() || tm.Mouse() != MouseNone {
		t.Fatal("modes were not reset")
	}
	tm = run(5, 1, "\x1b[?7labcdefg")
	want(t, tm, "abcdg")
}

func TestRepliesToQueries(t *testing.T) {
	tm := New(10, 5)
	var got strings.Builder
	tm.SetReply(func(b []byte) { got.Write(b) })
	tm.Write([]byte("\x1b[3;4H\x1b[6n\x1b[5n\x1b[c\x1b[>c"))
	if got.String() != "\x1b[3;4R\x1b[0n\x1b[?62;22c\x1b[>0;0;0c" {
		t.Fatalf("replies %q", got.String())
	}
	New(2, 2).Write([]byte("\x1b[6n")) // no reply set: dropped
}

func TestUntrustedStringsAreSkippedOrCleaned(t *testing.T) {
	tm := New(40, 2)
	var replies []byte
	tm.SetReply(func(b []byte) { replies = append(replies, b...) })
	tm.Write([]byte("a\x1b]52;c;SGVsbG8=\x07b\x1b]8;;http://x\x1b\\c\x1bP1$r\x1b\\d\x1b_Gi=1\x1b\\e"))
	want(t, tm, "abcde")
	tm.Write([]byte("\x1b]2;evil\x01\x00\x7f title\x07"))
	if tm.Title() != "evil title" {
		t.Fatalf("title %q", tm.Title())
	}
	if strings.ContainsAny(tm.Title(), "\x1b\x07\x00") {
		t.Fatalf("title kept control characters: %q", tm.Title())
	}
	tm.Write([]byte("\x1b]2;" + strings.Repeat("x", 10000) + "\x07"))
	if n := len([]rune(tm.Title())); n > 200 {
		t.Fatalf("title is %d runes", n)
	}
	if len(replies) != 0 {
		t.Fatalf("an untrusted string produced the reply %q", replies)
	}
}

func TestControlsInsideASequenceStillRun(t *testing.T) {
	tm := run(10, 3, "\x1b[2\n;3Hx")
	want(t, tm, "", "  x")
	tm = run(10, 1, "\x1b[3\x18A")
	want(t, tm, "A")
}

func TestREPAndIndex(t *testing.T) {
	tm := run(10, 2, "ab\x1b[3b")
	want(t, tm, "abbbb")
	tm = run(5, 3, "a\x1bDb\x1bEc")
	want(t, tm, "a", " b", "c")
}

func TestResizeKeepsTheCursorVisible(t *testing.T) {
	tm := run(5, 4, "1\r\n2\r\n3\r\n4")
	tm.Resize(5, 2)
	want(t, tm, "3", "4")
	if tm.Scrollback() != 2 {
		t.Fatalf("scrollback %d", tm.Scrollback())
	}
	if _, y, _ := tm.Cursor(); y != 1 {
		t.Fatalf("cursor row %d", y)
	}
	tm.Resize(8, 4)
	want(t, tm, "3", "4")
	if c, r := tm.Size(); c != 8 || r != 4 {
		t.Fatalf("size %dx%d", c, r)
	}
	tm.Resize(3, 4)
	tm.Write([]byte("\rxyzw"))
	want(t, tm, "3", "xyz", "w")
}

func TestScrollbackIsBounded(t *testing.T) {
	tm := New(4, 2)
	tm.SetScrollback(10)
	for i := 0; i < 100; i++ {
		tm.Write([]byte("x\r\n"))
	}
	if n := tm.Scrollback(); n < 10 || n > 13 {
		t.Fatalf("scrollback %d; want about 10", n)
	}
	tm.SetScrollback(0)
	if tm.Scrollback() != 0 {
		t.Fatalf("scrollback %d after SetScrollback(0)", tm.Scrollback())
	}
	tm.Write([]byte("a\r\nb\r\nc\r\n"))
	if tm.Scrollback() != 0 {
		t.Fatalf("scrollback %d with none kept", tm.Scrollback())
	}
	tm.Write([]byte("\x1b[3J"))
}

func TestResetRestoresTheScreen(t *testing.T) {
	tm := run(5, 2, "\x1b[31mabc\x1b[?25l\x1bc")
	want(t, tm)
	if _, _, vis := tm.Cursor(); !vis {
		t.Fatal("the cursor is hidden after a reset")
	}
	tm.Write([]byte("x"))
	if tm.Cell(0, 0).FG != nil {
		t.Fatal("the pen survived a reset")
	}
}

func TestNeverPanicsOnGarbage(t *testing.T) {
	tm := New(7, 3)
	junk := "\x1b[999999999;999999999H\x1b[-5A\x1b[;;;;;m\x1b[?9999h\x1b[1;1r\x1b[0;0r\x1b[99999@\x1b[99999P\x1b[99999L\x1b[99999M" +
		"\x1b[99999X\x1b[99999b\x1b[99999S\x1b[99999T\x1b[38;5m\x1b[38;2;1m\x1b[38:2:1:2m\x1b(\x1b)\x1b#8\x1b%G\x1b\x1b\x1b[\x1b]\x1b\x1bP\x00\x7f\xff\xfe"
	for i := 0; i < 3; i++ {
		tm.Write([]byte(junk))
		tm.Resize(1+i, 1+i)
		tm.Write([]byte(junk))
		tm.Row(5, 0)
		tm.Row(0, 99)
	}
}
