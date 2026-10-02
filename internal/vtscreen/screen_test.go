package vtscreen

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func write(s *Screen, text string) *Screen { s.Write([]byte(text)); return s }

func wantLines(t *testing.T, s *Screen, want ...string) {
	t.Helper()
	if got := s.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
}

func wantCursor(t *testing.T, s *Screen, x, y int, visible bool) {
	t.Helper()
	gx, gy, gv := s.Cursor()
	if gx != x || gy != y || gv != visible {
		t.Fatalf("cursor = (%d,%d,visible=%v), want (%d,%d,%v)", gx, gy, gv, x, y, visible)
	}
}

func TestPlainTextAndNewlines(t *testing.T) {
	s := write(NewScreen(6, 3), "ab\r\ncd")
	wantLines(t, s, "ab", "cd", "")
	wantCursor(t, s, 2, 1, true)
}

// A bare "\n" moves down without returning to column 0, as on a real terminal.
func TestLineFeedKeepsColumn(t *testing.T) {
	s := write(NewScreen(6, 3), "ab\ncd")
	wantLines(t, s, "ab", "  cd", "")
}

func TestCarriageReturnOverwrites(t *testing.T) {
	s := write(NewScreen(6, 1), "abcd\rXY")
	wantLines(t, s, "XYcd")
}

func TestControlBytesAreIgnored(t *testing.T) {
	s := write(NewScreen(6, 1), "a\x07\x08\x00b")
	wantLines(t, s, "ab")
}

func TestCursorMovement(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		x, y int
	}{
		{"home", "\x1b[H", 0, 0},
		{"absolute", "\x1b[3;4H", 3, 2},
		{"absolute row only", "\x1b[3H", 0, 2},
		{"absolute clamps", "\x1b[99;99H", 4, 3},
		{"up default", "\x1b[3;3H\x1b[A", 2, 1},
		{"up n clamps at top", "\x1b[2;2H\x1b[9A", 1, 0},
		{"down n clamps at bottom", "\x1b[9B", 0, 3},
		{"right n clamps at edge", "\x1b[9C", 4, 0},
		{"left n clamps at 0", "\x1b[3;3H\x1b[9D", 0, 2},
		{"zero count is one", "\x1b[0C", 1, 0},
		{"column absolute", "\x1b[4G", 3, 0},
		{"column absolute default", "\x1b[3C\x1b[G", 0, 0},
		{"column absolute clamps", "\x1b[99G", 4, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantCursor(t, write(NewScreen(5, 4), tc.seq), tc.x, tc.y, true)
		})
	}
}

func TestMovementThenWrite(t *testing.T) {
	s := write(NewScreen(5, 3), "\x1b[2;3Hxy")
	wantLines(t, s, "", "  xy", "")
}

func TestErase(t *testing.T) {
	fill := "aaaaa\r\nbbbbb\r\nccccc"
	tests := []struct {
		name string
		seq  string
		want []string
	}{
		{"line from cursor", "\x1b[2;3H\x1b[K", []string{"aaaaa", "bb", "ccccc"}},
		{"line explicit 0", "\x1b[2;3H\x1b[0K", []string{"aaaaa", "bb", "ccccc"}},
		{"whole line", "\x1b[2;3H\x1b[2K", []string{"aaaaa", "", "ccccc"}},
		{"screen from cursor", "\x1b[2;3H\x1b[J", []string{"aaaaa", "bb", ""}},
		{"whole screen", "\x1b[2J", []string{"", "", ""}},
		{"unsupported mode ignored", "\x1b[1K", []string{"aaaaa", "bbbbb", "ccccc"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantLines(t, write(NewScreen(5, 3), fill+tc.seq), tc.want...)
		})
	}
}

func TestAutowrapIsDeferred(t *testing.T) {
	s := write(NewScreen(3, 2), "abc")
	wantLines(t, s, "abc", "")
	wantCursor(t, s, 2, 0, true) // pending wrap: still on the last column
	write(s, "d")
	wantLines(t, s, "abc", "d")
	wantCursor(t, s, 1, 1, true)
}

func TestExplicitMoveCancelsPendingWrap(t *testing.T) {
	s := write(NewScreen(3, 2), "abc\x1b[1;1Hx")
	wantLines(t, s, "xbc", "")
}

func TestScrollsOffTheBottom(t *testing.T) {
	s := write(NewScreen(3, 2), "a\r\nb\r\nc")
	wantLines(t, s, "b", "c")
}

func TestAutowrapAtBottomScrolls(t *testing.T) {
	s := write(NewScreen(2, 2), "abcde")
	wantLines(t, s, "cd", "e")
}

func TestCursorVisibility(t *testing.T) {
	s := NewScreen(3, 1)
	wantCursor(t, s, 0, 0, true)
	write(s, "\x1b[?25l")
	wantCursor(t, s, 0, 0, false)
	write(s, "\x1b[?25h")
	wantCursor(t, s, 0, 0, true)
}

func TestOtherPrivateModesAreIgnored(t *testing.T) {
	s := write(NewScreen(5, 1), "\x1b[?1049h\x1b[?1000h\x1b[?2004hhi\x1b[?2004l")
	wantLines(t, s, "hi")
	wantCursor(t, s, 2, 0, true)
}

func TestUnknownSequencesAreIgnored(t *testing.T) {
	// A two-byte escape and a CSI with an unsupported final byte.
	wantLines(t, write(NewScreen(8, 1), "a\x1bMb\x1b[5Zc"), "abc")
}

// Control strings are skipped whole, however they end, so a hyperlink's URL or
// a window title never shows up as screen text.
func TestControlStringsAreSkipped(t *testing.T) {
	tests := []struct{ name, seq string }{
		{"osc bel", "\x1b]0;title\x07"},
		{"osc st", "\x1b]0;title\x1b\\"},
		{"osc 8 open", "\x1b]8;id=1;http://example.com\x07"},
		{"osc 8 close", "\x1b]8;;\x1b\\"},
		{"dcs", "\x1bPq#0;2;0;0;0\x1b\\"},
		{"apc", "\x1b_Gi=1;AAAA\x1b\\"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantLines(t, write(NewScreen(12, 1), "a"+tc.seq+"b"), "ab")
		})
	}
}

func TestHyperlinkedTextKeepsOnlyTheText(t *testing.T) {
	s := write(NewScreen(10, 1), "\x1b]8;;http://x\x07go\x1b]8;;\x07!")
	wantLines(t, s, "go!")
	wantCursor(t, s, 3, 0, true)
}

func TestControlStringSplitAcrossWrites(t *testing.T) {
	text := "a\x1b]8;;http://x\x1b\\b"
	for cut := 1; cut < len(text); cut++ {
		s := NewScreen(10, 1)
		s.Write([]byte(text[:cut]))
		s.Write([]byte(text[cut:]))
		if got := s.Lines()[0]; got != "ab" {
			t.Fatalf("cut at %d: line = %q, want %q", cut, got, "ab")
		}
	}
}

func TestSplitWritesReassemble(t *testing.T) {
	text := "\x1b[1;31mhé\x1b[0m\x1b[2;1Hz"
	want := write(NewScreen(5, 2), text)
	for cut := 1; cut < len(text); cut++ {
		s := NewScreen(5, 2)
		s.Write([]byte(text[:cut]))
		s.Write([]byte(text[cut:]))
		if !reflect.DeepEqual(s.Lines(), want.Lines()) || s.Cell(0, 0) != want.Cell(0, 0) {
			t.Fatalf("cut at %d: lines %q, want %q", cut, s.Lines(), want.Lines())
		}
	}
}

func TestSGRAttributes(t *testing.T) {
	s := write(NewScreen(10, 1), "\x1b[1;3;4;7;9mA\x1b[22;23;24;27;29mB\x1b[2mC\x1b[0mD")
	a := s.Cell(0, 0)
	if !(a.Bold && a.Italic && a.Underline && a.Reverse && a.Strike) || a.Dim {
		t.Fatalf("A = %+v", a)
	}
	if b := s.Cell(1, 0); b != (Cell{Rune: 'B'}) {
		t.Fatalf("B = %+v, want plain", b)
	}
	if c := s.Cell(2, 0); !c.Dim || c.Bold {
		t.Fatalf("C = %+v", c)
	}
	if d := s.Cell(3, 0); d != (Cell{Rune: 'D'}) {
		t.Fatalf("D = %+v, want plain", d)
	}
}

func TestSGRResetForms(t *testing.T) {
	for _, seq := range []string{"\x1b[m", "\x1b[0m", "\x1b[;m"} {
		s := write(NewScreen(3, 1), "\x1b[1m"+seq+"x")
		if c := s.Cell(0, 0); c != (Cell{Rune: 'x'}) {
			t.Fatalf("%q: cell = %+v, want plain", seq, c)
		}
	}
}

func TestSGRColours(t *testing.T) {
	tests := []struct {
		name   string
		seq    string
		fg, bg ansi.Color
	}{
		{"basic", "\x1b[31;42m", ansi.BasicColor(1), ansi.BasicColor(2)},
		{"bright", "\x1b[91;102m", ansi.BasicColor(9), ansi.BasicColor(10)},
		{"256", "\x1b[38;5;200;48;5;17m", ansi.Color256(200), ansi.Color256(17)},
		{"truecolor", "\x1b[38;2;1;2;3;48;2;4;5;6m", ansi.RGB{R: 1, G: 2, B: 3}, ansi.RGB{R: 4, G: 5, B: 6}},
		{"clamped", "\x1b[38;2;300;-5;7m", ansi.RGB{R: 255, G: 0, B: 7}, nil},
		{"default restores", "\x1b[31;41m\x1b[39;49m", nil, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := write(NewScreen(3, 1), tc.seq+"x").Cell(0, 0)
			if !reflect.DeepEqual(c.FG, tc.fg) || !reflect.DeepEqual(c.BG, tc.bg) {
				t.Fatalf("fg=%v bg=%v, want fg=%v bg=%v", c.FG, c.BG, tc.fg, tc.bg)
			}
		})
	}
}

func TestUnderlineStyleSubparameter(t *testing.T) {
	s := write(NewScreen(3, 1), "\x1b[4:3mA\x1b[4:0mB")
	if !s.Cell(0, 0).Underline || s.Cell(1, 0).Underline {
		t.Fatalf("A=%+v B=%+v", s.Cell(0, 0), s.Cell(1, 0))
	}
}

func TestEraseDoesNotCarryPenAttributes(t *testing.T) {
	s := write(NewScreen(3, 1), "\x1b[41mabc\x1b[2K")
	if c := s.Cell(0, 0); c != (Cell{Rune: ' '}) {
		t.Fatalf("erased cell = %+v, want blank default", c)
	}
}

func TestCellOffScreenIsBlank(t *testing.T) {
	s := write(NewScreen(2, 2), "ab")
	for _, p := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}} {
		if c := s.Cell(p[0], p[1]); c != (Cell{Rune: ' '}) {
			t.Fatalf("Cell(%d,%d) = %+v", p[0], p[1], c)
		}
	}
}

func TestResizeCropsAndExtendsWithoutReflow(t *testing.T) {
	s := write(NewScreen(4, 2), "abcd\r\nefgh")
	s.Resize(2, 3)
	wantLines(t, s, "ab", "ef", "")
	s.Resize(4, 2)
	wantLines(t, s, "ab", "ef")
}

func TestResizeClampsCursor(t *testing.T) {
	s := write(NewScreen(5, 5), "\x1b[5;5H")
	s.Resize(3, 2)
	wantCursor(t, s, 2, 1, true)
}

func TestResizeReflowRewrapsSoftLines(t *testing.T) {
	s := write(NewScreen(6, 3), "abcdef"+"gh")
	// "abcdefgh" autowrapped over two rows at width 6.
	wantLines(t, s, "abcdef", "gh", "")
	s.ResizeReflow(4, 3)
	wantLines(t, s, "abcd", "efgh", "")
	s.ResizeReflow(8, 3)
	wantLines(t, s, "abcdefgh", "", "")
}

func TestResizeReflowKeepsHardLines(t *testing.T) {
	s := write(NewScreen(6, 3), "ab\r\ncd")
	s.ResizeReflow(3, 3)
	wantLines(t, s, "ab", "cd", "")
	wantCursor(t, s, 2, 1, true)
}

func TestResizeReflowScrollsOverflowOffTheTop(t *testing.T) {
	s := write(NewScreen(8, 3), "aaaaaaaa"+"bbbbbbbb"+"cc")
	s.ResizeReflow(4, 3)
	got := s.Lines()
	if got[len(got)-1] != "cc" || got[0] != "bbbb" {
		t.Fatalf("lines = %q", got)
	}
}

func TestResizeReflowDegenerateSizeFallsBack(t *testing.T) {
	s := write(NewScreen(4, 2), "ab")
	s.ResizeReflow(0, 0) // must not panic
	s.Write([]byte("x")) // writes to a zero-size screen are dropped
}

func TestZeroSizeScreenDropsOutput(t *testing.T) {
	s := NewScreen(0, 0)
	s.Write([]byte("hello\r\n"))
	if got := s.Lines(); len(got) != 0 {
		t.Fatalf("lines = %q", got)
	}
}
